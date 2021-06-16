/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package cmd

import (
	"cli/callx"
	"encoding/json"
	"fmt"
	"model"
	"path"
	"strconv"
	"strings"
	"time"
	"xbase/common"

	"github.com/spf13/cobra"
)

func NewMysqlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mysql <subcommand>",
		Short: "mysql related commands",
	}

	cmd.AddCommand(NewMysqlStartMonitorCommand())
	cmd.AddCommand(NewMysqlStopMonitorCommand())
	cmd.AddCommand(NewMysqlStartCommand())
	cmd.AddCommand(NewMysqlShutDownCommand())
	cmd.AddCommand(NewMysqlRebuildMeCommand())
	cmd.AddCommand(NewMysqlDoBackupCommand())
	cmd.AddCommand(NewMysqlCancelBackupCommand())
	cmd.AddCommand(NewMysqlCreateUserCommand())
	cmd.AddCommand(NewMysqlCreateSuperUserCommand())
	cmd.AddCommand(NewMysqlDropUserCommand())
	cmd.AddCommand(NewMysqlChangePasswordCommand())
	cmd.AddCommand(NewMysqlSetVarCommand())
	cmd.AddCommand(NewMysqlKillCommand())
	cmd.AddCommand(NewMysqlStatusCommand())
	cmd.AddCommand(NewMysqlCreateUserWithPrivilegesCommand())
	cmd.AddCommand(NewMysqlGetUserCommand())

	return cmd
}

// stop monitor
func NewMysqlStopMonitorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stopmonitor",
		Short: "stop mysqld monitor",
		Run:   mysqlStopMonitorCommandFn,
	}

	return cmd
}

func mysqlStopMonitorCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}
	log.Warning("prepare.to.stop.mysql.monitor")
	conf, err := GetConfig()
	ErrorOK(err)
	rsp, err := callx.StopMonitorRPC(conf.Server.Endpoint)
	ErrorOK(err)
	RspOK(rsp.RetCode)
	log.Warning("prepare.to.stop.mysql.monitor.done")
}

// start monitor
func NewMysqlStartMonitorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "startmonitor",
		Short: "start mysqld monitor",
		Run:   mysqlStartMonitorCommandFn,
	}

	return cmd
}

func mysqlStartMonitorCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}
	log.Warning("prepare.to.start.mysql.monitor")

	conf, err := GetConfig()
	ErrorOK(err)
	rsp, err := callx.StartMonitorRPC(conf.Server.Endpoint)
	ErrorOK(err)
	RspOK(rsp.RetCode)
	log.Warning("prepare.to.start.mysql.monitor.done")
}

// start mysql
func NewMysqlStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start mysql",
		Run:   mysqlStartCommandFn,
	}

	return cmd
}

func mysqlStartCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}

	log.Warning("prepare.to.start.mysql")
	conf, err := GetConfig()
	ErrorOK(err)
	callx.StartMysqldRPC(conf.Server.Endpoint)
	log.Warning("start.mysql.done")
}

// shutdown mysql
func NewMysqlShutDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "shutdown",
		Run: mysqlShutDownCommandFn,
	}

	return cmd
}

func mysqlShutDownCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}

	log.Warning("prepare.to.shutdown.mysql")

	conf, err := GetConfig()
	ErrorOK(err)

	// shutdown
	callx.ShutdownMysqldRPC(conf.Server.Endpoint)

	// wait mysqld shutdown
	callx.WaitMysqldShutdownRPC(conf.Server.Endpoint)
	log.Warning("shutdown.mysql.done")
}

// rebuild me
var (
	fromStr string
	force   bool
)

func NewMysqlRebuildMeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rebuildme [--from=endpoint][--force]",
		Short: "rebuild a slave --from=endpoint --force",
		Run:   mysqlRebuildMeCommandFn,
	}
	cmd.Flags().StringVar(&fromStr, "from", "", "--from=endpoint")
	cmd.Flags().BoolVar(&force, "force", false, "--force")

	return cmd
}

func getLocalTrxCount(self string, bestone string) (int, error) {
	count := 0

	rsp1, err := callx.GetGTIDRPC(bestone)
	if err != nil {
		return -1, fmt.Errorf("get.gtid.from.bestone[%v].failed[%v]", bestone, err)
	} else if rsp1.GTID.Executed_GTID_Set == "" {
		return -1, fmt.Errorf("the.Executed_GTID_Set.of.bestone[%v].is.null", bestone)
	}
	fromGTID := rsp1.GTID

	rsp1, err = callx.GetGTIDRPC(self)
	if err != nil {
		return -1, fmt.Errorf("get.gtid.from.myself[%v].failed[%v]", self, err)
	} else if rsp1.GTID.Executed_GTID_Set == "" {
		return -1, fmt.Errorf("the.Executed_GTID_Set.of.myself[%v].is.null", self)
	}
	localGTID := rsp1.GTID

	rsp2, err := callx.GetGTIDSubtractRPC(self, localGTID.Executed_GTID_Set, fromGTID.Executed_GTID_Set)
	if err != nil {
		return -1, fmt.Errorf("get.gtid.subtract.from.self[%v].failed[%v]", self, err)
	}
	subtract := rsp2.Subtract

	// compute the number of local transactions
	gtidSet := strings.Split(subtract, "\n")
	for _, gtid := range gtidSet {
		gtid = strings.TrimSpace(gtid)
		gtid = strings.TrimSuffix(gtid, ",")
		diffs := strings.Split(gtid, ":")[1:]
		for _, diff := range diffs {
			values := strings.Split(diff, "-")
			if len(values) == 1 {
				count += 1
			} else {
				s, _ := strconv.Atoi(values[0])
				e, _ := strconv.Atoi(values[1])
				count += e - s + 1
			}
		}
	}

	return count, nil
}

func mysqlRebuildMeCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}

	log.Warning(`=====prepare.to.rebuildme=====
			IMPORTANT: Please check that the backup run completes successfully.
			           At the end of a successful backup run innobackupex
			           prints "completed OK!".
			`)

	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	bestone := ""
	version := strings.TrimSpace(conf.Mysql.Version)
	datadir := conf.Backup.BackupDir
	binlogDir := ""
	binlogPrefix := ""
	maxAllowedLocalTrxCount := conf.Backup.MaxAllowedLocalTrxCount

	// 1. first to check I am leader or not
	{
		log.Warning("S1-->check.raft.leader")
		leader, err := callx.GetClusterLeader(self)
		ErrorOK(err)
		if leader == self {
			log.Panic("I[%v].am.leader.you.cant.rebuildme.sir", self)
		}
	}

	// 2. find the best to backup and check gtid
	{
		if fromStr != "" {
			bestone = fromStr
		} else {
			bestone, err = callx.FindBestoneForBackup(self)
			ErrorOK(err)
		}
		log.Warning("S2-->prepare.rebuild.from[%v]....", bestone)

		// check if there are more than 2 local transactions on the local node than bestone
		if force {
			log.Warning("S2-->the.[--force].is.specified.skip.check.gtid")
		} else {
			working, err := callx.MysqlIsWorkingRPC(self)
			ErrorOK(err)
			if working == false {
				log.Panic("local.mysql.is.not.working.you.cant.rebuildme.sir")
			}
			localTrxCount, err := getLocalTrxCount(self, bestone)
			ErrorOK(err)
			if localTrxCount > maxAllowedLocalTrxCount {
				log.Panic("I[%v].have.[%v].local.transactions.more.than.maxAllowedLocalTrxCount[%v].compared.to.from[%v].you.cant.rebuildme.sir", self, localTrxCount, maxAllowedLocalTrxCount, bestone)
			}
		}
	}

	// 3. check bestone is not in BACKUPING
	{
		rsp, err := callx.GetMysqldStatusRPC(bestone)
		ErrorOK(err)
		if rsp.BackupStatus == model.MYSQLD_BACKUPING {
			log.Warning("S3-->check.bestone[%v].is.backuping....", bestone)
			log.Panic("bestone[%v].is.backuping....", bestone)
		}
		log.Warning("S3-->check.bestone[%v].is.OK....", bestone)
	}

	// 4. set learner
	{
		log.Warning("S4-->set.learner")
		if _, err := callx.SetLearnerRPC(self); err != nil {
			log.Error("SetLearnerRPC.error[%v]", err)
		}
	}

	// 5. stop monitor
	{
		log.Warning("S5-->stop.monitor")
		callx.StopMonitorRPC(self)
	}

	// 6. force kill mysqld
	{
		log.Warning("S6-->kill.mysql")
		err := callx.KillMysqldRPC(self)
		ErrorOK(err)

		// wait
		err = callx.WaitMysqldShutdownRPC(self)
		ErrorOK(err)

		// set the mysql state to dead, avoid failure to rebuild node with small amounts of data
		err = callx.SetMysqlStateRPC(self, model.MysqlDead)
		ErrorOK(err)
	}

	// 7. check bestone is not in BACKUPING again
	{
		rsp, err := callx.GetMysqldStatusRPC(bestone)
		ErrorOK(err)
		if rsp.BackupStatus == model.MYSQLD_BACKUPING {
			log.Warning("S7-->check.bestone[%v].is.backuping....", bestone)
			log.Panic("bestone[%v].is.backuping....", bestone)
		}
		log.Warning("S7-->check.bestone[%v].is.OK....", bestone)
	}

	// 8. remove data files
	{
		// remove mysql data
		cmds := "bash"
		args := []string{
			"-c",
			fmt.Sprintf("rm -rf %s/*", datadir),
		}

		_, err := common.RunCommand(cmds, args...)
		ErrorOK(err)
		log.Warning("S8-->clear.datadir[%v]", datadir)

		/*
			Remove mysql binlog and index, considering that mysql binlog or index may not be in the same directory as the data.
			For example, The contents of file my.cnf are as follows:

			#log-bin=/data/mysql-log/mysql-bin/mysql-bin
			log-bin=./mysql-bin
			log-bin=/data/mysql-log/mysql-bin/mysql-bin
			#log-bin=/data/mysql/mysql-bin
			log-bin-index=/data/mysql/mysql-bin.index
			log-bin-index=/data/mysql-log/mysql-bin/mysql-bin.index
			log-bin-index=./mysql-bin.index
			#log-bin-index=/data/mysql-log/mysql-bin/mysql-bin.index

			The following shell instruction resolves that the paths of log-bin and log-bin-index are
			/data/mysql-log/mysql-bin/mysql-bin and ./mysql-bin.index respectively.
		*/
		args = []string{
			"-c",
			fmt.Sprintf("grep 'log-bin=' %s | sed -r '/^#/d' | awk -F '=' '{print $2}' | tail -n 1", conf.Mysql.DefaultsFile),
		}
		binlogPrefix, err = common.RunCommand(cmds, args...)
		ErrorOK(err)
		binlogPrefix = strings.TrimSpace(binlogPrefix)
		if binlogPrefix != "" && strings.Index(binlogPrefix, "/") == 0 {
			binlogDir = path.Dir(binlogPrefix)
			if binlogDir != path.Dir(datadir+"/") {
				log.Warning("mysql.binlog.dir[%v].is.different.from.data.dir[%v]", binlogDir, datadir)
				args = []string{
					"-c",
					fmt.Sprintf("rm -f %s/*", binlogDir),
				}

				_, err := common.RunCommand(cmds, args...)
				ErrorOK(err)
				log.Warning("S8-->clear.mysql.binlog[%v.*]", binlogPrefix)
			}
		}

		args = []string{
			"-c",
			fmt.Sprintf("grep 'log-bin-index=' %s | sed -r '/^#/d' | awk -F '=' '{print $2}' | tail -n 1", conf.Mysql.DefaultsFile),
		}
		indexPath, err := common.RunCommand(cmds, args...)
		ErrorOK(err)
		if indexPath != "" && strings.Index(indexPath, "/") == 0 {
			indexDir := path.Dir(indexPath)
			if indexDir != path.Dir(datadir+"/") {
				log.Warning("mysql.binlog.index[%v].is.not.in.data.dir[%v]", indexPath, datadir)
				args = []string{
					"-c",
					fmt.Sprintf("rm -f %s", indexPath),
				}
				_, err := common.RunCommand(cmds, args...)
				ErrorOK(err)
				log.Warning("S8-->clear.mysql.binlog.index[%v]", indexPath)
			}
		}
	}

	// 9. do backup from bestone
	{
		log.Warning("S9-->xtrabackup.begin....")
		rsp, err := callx.RequestBackupRPC(bestone, conf, datadir)
		ErrorOK(err)
		RspOK(rsp.RetCode)
		log.Warning("S9-->xtrabackup.end....")
	}

	// 10. do apply-log
	{
		log.Warning("S10-->apply-log.begin....")
		err := callx.DoApplyLogRPC(conf.Server.Endpoint, datadir)
		ErrorOK(err)

		if version == "mysql80" {
			/*
				For 5.7, mysql will not work properly if log-bin-index is specified and log-bin is not specified.
				But For 8.0, it works fine, mysql will automatically generate a new file based on the current serial number.

				Xtrabackup will copy the nearest binlog from the source to the current data directory,
				therefore, you need to move the last binlog to the directory specified by log-bin.
			*/
			if binlogDir != "" && strings.Index(binlogDir, "/") == 0 {
				datadir2 := path.Dir(datadir + "/")
				// if the binlog path is absolute and different from mysql data directory, move the binlog
				if binlogDir != datadir2 {
					log.Warning("mysql.binlog.dir[%v].is.different.from.data.dir[%v]", binlogDir, datadir2)
					binlogBase := path.Base(binlogPrefix)
					cmds := "bash"
					args = []string{
						"-c",
						fmt.Sprintf("mv %s/%s.* %s", datadir2, binlogBase, binlogDir),
					}
					_, err := common.RunCommand(cmds, args...)
					ErrorOK(err)
					log.Warning("move.binlog[%v/%v.*].to.dir[%v]", datadir2, binlogBase, binlogDir)
				}
			}
		}
		log.Warning("S10-->apply-log.end....")
	}
	
        //Pre 11 change datadir owner ->mysql. patch by wubx
	{
		datadir := conf.Backup.BackupDir
		cmds :="bash"
		args := []string{
			"-c",
			fmt.Sprintf("chown -R mysql:mysql %s", datadir),
		}
		_, err := common.RunCommand(cmds, args...)
		ErrorOK(err)
		log.Warning("Pre-S11-->chown owner to mysql[%v]", datadir)
	}

	// 11. start mysqld
	{
		log.Warning("S11-->start.mysql.begin...")
		if _, err := callx.StartMonitorRPC(self); err != nil {
			log.Error("start.mysql..error[%v]", err)
		}
		log.Warning("S11-->start.mysql.end...")
	}

	// 12. wait mysqld running
	{
		log.Warning("S12-->wait.mysqld.running.begin....")
		callx.WaitMysqldRunningRPC(self)
		log.Warning("S12-->wait.mysqld.running.end....")
	}

	// 13. wait mysql working
	{
		log.Warning("S13-->wait.mysql.working.begin....")
		callx.WaitMysqlWorkingRPC(self)
		log.Warning("S13-->wait.mysql.working.end....")
	}

	// 14. stop slave and reset slave all
	{
		log.Warning("S14-->stop.and.reset.slave.begin....")
		if _, err := callx.MysqlResetSlaveAllRPC(self); err != nil {
			log.Error("mysql.stop.adn.reset.slave.error[%v]", err)
		}
		log.Warning("S14-->stop.and.reset.slave.end....")
	}

	// 15. set gtid_purged
	{
		log.Warning("S15-->reset.master.begin....")
		if version == "mysql80" {
			log.Warning("S15-->reset.master.skip.mysql80")
		} else {
			callx.MysqlResetMasterRPC(self)
			log.Warning("S15-->reset.master.end....")

			gtid, err := callx.GetXtrabackupGTIDPurged(self, datadir)
			ErrorOK(err)

			log.Warning("S15-->set.gtid_purged[%v].begin....", gtid)
			rsp, err := callx.SetGlobalVarRPC(self, fmt.Sprintf("SET GLOBAL gtid_purged='%s'", gtid))
			ErrorOK(err)
			RspOK(rsp.RetCode)
			log.Warning("S15-->set.gtid_purged.end....")
		}
	}

	// 16. enable raft
	{
		// check whether the state is IDLE or not
		if conf.Raft.SuperIDLE {
			log.Warning("S16-->disable.raft.again...")
			if _, err := callx.DisableRaftRPC(self); err != nil {
				log.Error("enbleRaftRPC.error[%v]", err)
			}
			log.Warning("S16-->run.as.IDLE...")
		} else {
			log.Warning("S16-->enable.raft.begin...")
			if _, err := callx.EnableRaftRPC(self); err != nil {
				log.Error("enbleRaftRPC.error[%v]", err)
			}
			log.Warning("S16-->enable.raft.done...")
		}
	}

	// 17. wait change to master
	{
		log.Warning("S17-->wait[%v ms].change.to.master...", conf.Raft.ElectionTimeout)
		time.Sleep(time.Duration(conf.Raft.ElectionTimeout))
	}

	// 18. start slave
	{
		log.Warning("S18-->start.slave.begin....")
		if _, err := callx.MysqlStartSlaveRPC(self); err != nil {
			log.Error("mysql.start.slave.error[%v]", err)
		} else {
			log.Warning("S18-->start.slave.end....")
		}
	}

	log.Warning("completed OK!")
	log.Warning("rebuildme.all.done....")
}

var (
	toStr string
)

func NewMysqlDoBackupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup --to=backupdir",
		Short: "backup this mysql to backupdir",
		Run:   mysqlDoBackupCommandFn,
	}
	cmd.Flags().StringVar(&toStr, "to", "", "--to=backupdir")

	return cmd
}

func mysqlDoBackupCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 || len(toStr) == 0 {
		cmd.Usage()
		ErrorOK(fmt.Errorf("args.must.be: --to=backupdir"))
	}

	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	bestone := ""

	// 1. find the best to backup
	{
		node, err := callx.FindBestoneForBackup(self)
		ErrorOK(err)
		bestone = node
		log.Warning("S1-->found.the.best.backup.host[%v]....", bestone)
	}

	backupdir := toStr
	// 2. remove and make backupdir files
	{
		cmds := "bash"
		args := []string{
			"-c",
			fmt.Sprintf("rm -rf %s ; mkdir %s", backupdir, backupdir),
		}

		_, err := common.RunCommand(cmds, args...)
		ErrorOK(err)
		log.Warning("S2-->rm.and.mkdir.backupdir[%v]", backupdir)
	}

	// 3. do backup from bestone
	{
		log.Warning("S3-->xtrabackup.begin....")
		rsp, err := callx.RequestBackupRPC(bestone, conf, backupdir)
		ErrorOK(err)
		RspOK(rsp.RetCode)
		log.Warning("S3-->xtrabackup.end....")
	}

	// 4. do apply-log
	{
		log.Warning("S4-->apply-log.begin....")
		err := callx.DoApplyLogRPC(conf.Server.Endpoint, backupdir)
		ErrorOK(err)
		log.Warning("S4-->apply-log.end....")
	}

	log.Warning("completed OK!")
	log.Warning("backup.all.done....")
}

func NewMysqlCancelBackupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cancelbackup",
		Run: mysqlCancelBackupCommandFn,
	}

	return cmd
}

func mysqlCancelBackupCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		ErrorOK(fmt.Errorf("too.many.args"))
	}

	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	{
		log.Warning("backup.cancel.begin....")
		rsp, err := callx.BackupCancelRPC(self)
		ErrorOK(err)
		RspOK(rsp.RetCode)
		log.Warning("backup.cancel.done....")
	}
}

// create normal user
func NewMysqlCreateUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "createuser <user> <host> <password> <YES | NO>",
		Short: "create mysql normal user",
		Run:   mysqlCreateUserCommandFn,
	}

	return cmd
}

func mysqlCreateUserCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		ErrorOK(fmt.Errorf("args.count.error:should.be.4"))
	}

	user := args[0]
	host := args[1]
	passwd := args[2]
	ssl := args[3]
	log.Warning("prepare.to.create.normaluser[%v]@[%v]", user, host)
	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	{
		leader, err := callx.GetClusterLeader(self)
		ErrorOK(err)
		rsp, err := callx.CreateNormalUserRPC(leader, user, host, passwd, ssl)
		ErrorOK(err)
		RspOK(rsp.RetCode)
	}
	log.Warning("create.normaluser[%v].done", user)
}

// create super user
func NewMysqlCreateSuperUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "createsuperuser <user> <host> <password> <YES | NO>",
		Short: "create mysql super user",
		Run:   mysqlCreateSuperUserCommandFn,
	}
	return cmd
}

func mysqlCreateSuperUserCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 4 {
		ErrorOK(fmt.Errorf("args.count.should.be.4: <user> <host> <password> <YES | NO>"))
	}

	user := args[0]
	host := args[1]
	passwd := args[2]
	ssl := args[3]
	log.Warning("prepare.to.create.superuser[%v]@[%v].with.ssl[%v]", user, host, ssl)
	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	{
		leader, err := callx.GetClusterLeader(self)
		ErrorOK(err)
		rsp, err := callx.CreateSuperUserRPC(leader, user, host, passwd, ssl)
		ErrorOK(err)
		RspOK(rsp.RetCode)
	}
	log.Warning("create.superuser[%v].done", user)
}

// drop user(normal&super)
func NewMysqlDropUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dropuser <user> <host>",
		Short: "drop mysql user",
		Run:   mysqlDropUserCommandFn,
	}

	return cmd
}

func mysqlDropUserCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		ErrorOK(fmt.Errorf("args.count.error:should.be.2"))
	}

	user := args[0]
	host := args[1]
	log.Warning("prepare.to.drop.user[%v]@[%v]", user, host)
	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	{
		leader, err := callx.GetClusterLeader(self)
		ErrorOK(err)
		rsp, err := callx.DropUserRPC(leader, user, host)
		ErrorOK(err)
		RspOK(rsp.RetCode)
	}
	log.Warning("drop.user[%v].done", user)
}

// change normal user password
func NewMysqlChangePasswordCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changepassword <user> <host> <password>",
		Short: "update mysql user password",
		Run:   mysqlChangePasswordCommandFn,
	}

	return cmd
}

func mysqlChangePasswordCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		ErrorOK(fmt.Errorf("args.count.error:should.be.3"))
	}

	user := args[0]
	host := args[1]
	passwd := args[2]
	log.Warning("prepare.to.changepassword.user[%v]@[%v]", user, host)
	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	{
		leader, err := callx.GetClusterLeader(self)
		ErrorOK(err)

		rsp, err := callx.ChangeUserPasswordRPC(leader, user, host, passwd)
		ErrorOK(err)
		RspOK(rsp.RetCode)
	}
	log.Warning("changepassword.user[%v].done", user)
}

// set global sysvar
func NewMysqlSetVarCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sysvar",
		Short: "set global variables",
		Run:   mysqlSetGlobalVarCommandFn,
	}

	return cmd
}

func mysqlSetGlobalVarCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ErrorOK(fmt.Errorf("args.should.be.1"))
	}
	log.Warning("prepare.to.set.global.var[%v]", args[0])
	conf, err := GetConfig()
	ErrorOK(err)
	rsp, err := callx.SetGlobalVarRPC(conf.Server.Endpoint, args[0])
	ErrorOK(err)
	RspOK(rsp.RetCode)
	log.Warning("set.global.var[%v].done", args[0])
}

// kill mysql
func NewMysqlKillCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kill",
		Short: "kill mysql pid(becareful!)",
		Run:   mysqlKillCommandFn,
	}

	return cmd
}

func mysqlKillCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}
	log.Warning("prepare.to.kill.mysql")
	conf, err := GetConfig()
	ErrorOK(err)
	err = callx.KillMysqldRPC(conf.Server.Endpoint)
	ErrorOK(err)
	log.Warning("prepare.to.kill.mysql.done")
}

// mysql status api format in JSON
func NewMysqlStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "mysql status in JSON(mysqld/slave_SQL/IO is running)",
		Run:   mysqlStatusCommandFn,
	}

	return cmd
}

func mysqlStatusCommandFn(cmd *cobra.Command, args []string) {
	type Status struct {
		Slave_io_running      bool   `json:"slave_io_running"`
		Slave_sql_running     bool   `json:"slave_sql_running"`
		Mysqldrunning         bool   `json:"mysqld_running"`
		Mysqlworking          bool   `json:"mysql_working"`
		Seconds_behind_master string `json:"seconds_behind_master"`
		Last_error            string `json:"last_error"`
		Monitor               string `json:"monitor"`
	}
	status := &Status{}

	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}
	conf, err := GetConfig()
	self := conf.Server.Endpoint
	ErrorOK(err)
	// mysqld info
	if running, err := callx.MysqldIsRunningRPC(self); err == nil {
		status.Mysqldrunning = running
		// slave info
		if running {
			rsp, err := callx.GetGTIDRPC(self)
			ErrorOK(err)

			status.Slave_io_running = rsp.GTID.Slave_IO_Running
			status.Slave_sql_running = rsp.GTID.Slave_SQL_Running
			status.Seconds_behind_master = rsp.GTID.Seconds_Behind_Master
			status.Last_error = rsp.GTID.Last_Error

			mysqlworking, err := callx.MysqlIsWorkingRPC(self)
			ErrorOK(err)
			status.Mysqlworking = mysqlworking
		}
	}

	if rsp, err := callx.GetMysqldStatusRPC(self); err == nil {
		status.Monitor = rsp.MonitorInfo
	}

	statusB, _ := json.Marshal(status)
	fmt.Printf("%s", string(statusB))
}

var (
	grantUser     string
	grantPasswd   string
	grantDatabase string
	grantTable    string
	grantHost     string
	grantPrivs    string
	requireSSL    string
)

// create normal user with privileges
func NewMysqlCreateUserWithPrivilegesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "createuserwithgrants",
		Short: "create mysql normal user with privileges",
		Run:   mysqlCreateUserWithPrivilegesCommandFn,
	}
	cmd.Flags().StringVar(&grantUser, "user", "", "--user=<user>")
	cmd.Flags().StringVar(&grantPasswd, "passwd", "", "--passwd=<passwd>")
	cmd.Flags().StringVar(&grantDatabase, "database", "", "--database=<database>")
	cmd.Flags().StringVar(&grantTable, "table", "", "--table=<table>")
	cmd.Flags().StringVar(&grantHost, "host", "", "--host=<host>")
	cmd.Flags().StringVar(&grantPrivs, "privs", "for example:SELECT,CREATE(comma-separated)", "--privs=<privs>")
	cmd.Flags().StringVar(&requireSSL, "ssl", "NO", "--ssl=<YES | NO>")

	return cmd
}

func mysqlCreateUserWithPrivilegesCommandFn(cmd *cobra.Command, args []string) {
	log.Warning("prepare.to.create.normaluser[%v]@[%v].with.privs", grantUser, grantHost)
	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	{
		leader, err := callx.GetClusterLeader(self)
		ErrorOK(err)
		rsp, err := callx.CreateUserWithPrivRPC(leader, grantUser, grantPasswd, grantDatabase, grantTable, grantHost, grantPrivs, requireSSL)
		ErrorOK(err)
		RspOK(rsp.RetCode)
	}
	log.Warning("create.normaluser[%v].with.privs.done", grantUser)
}

// NewMysqlGetUserCommand get mysql user list api (format in JSON)
func NewMysqlGetUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "getuser",
		Short: "get mysql user list",
		Run:   mysqlGetUserCommandFn,
	}

	return cmd
}

func mysqlGetUserCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}

	conf, err := GetConfig()
	ErrorOK(err)

	self := conf.Server.Endpoint
	{
		leader, err := callx.GetClusterLeader(self)
		ErrorOK(err)
		rsp, err := callx.GetMysqlUserRPC(leader)
		ErrorOK(err)
		RspOK(rsp.RetCode)
		Users, _ := json.Marshal(rsp.Users)
		fmt.Printf("%s\n", string(Users))
	}
}
