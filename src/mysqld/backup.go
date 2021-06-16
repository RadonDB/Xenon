/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package mysqld

import (
	"config"
	"fmt"
	"model"
	"strings"
	"time"
	"xbase/common"
	"xbase/xlog"

	"github.com/pkg/errors"
)

const (
	// backupOk used to completed of xtrabackup
	backupOk           = "completed OK!"
	backupOkCheckTimes = 1
)

// Backup tuple.
type Backup struct {
	log    *xlog.Log
	conf   *config.BackupConfig
	cmd    common.Command
	start  time.Time
	status model.MYSQLD_STATUS
	stats  model.BackupStats
}

// NewBackup creates new backup tuple.
func NewBackup(conf *config.BackupConfig, log *xlog.Log) *Backup {
	return &Backup{
		conf:   conf,
		log:    log,
		cmd:    common.NewLinuxCommand(log),
		status: model.MYSQLD_BACKUPNONE,
	}
}

// SetCMDHandler used to set the command handler.
func (b *Backup) SetCMDHandler(h common.Command) {
	b.cmd = h
}

// check ssh tunnel with password
func (b *Backup) checkSSHTunnelWithPass(req *model.BackupRPCRequest) bool {
	log := b.log
	args := []string{
		"-c",
		"sshpass -V",
	}

	// check sshpass
	if outs, err := b.cmd.RunCommand(bash, args); err != nil {
		log.Error("sshpass.not.installed.check.outs[%+v:%v]", err, outs)
		return false
	}

	args = []string{
		"-c",
		fmt.Sprintf("sshpass -p %s ssh -o 'StrictHostKeyChecking no' %s@%s -p %d 'echo 1'", req.SSHPasswd, req.SSHUser, req.SSHHost, req.SSHPort),
	}

	outs, err := b.cmd.RunCommand(bash, args)
	if err != nil {
		log.Error("ssh.tunnel[sshpass].error.outs[%+v:%v]", err, outs)
		return false
	}
	return true
}

// check ssh tunnel with key
func (b *Backup) checkSSHTunnelWithKey(req *model.BackupRPCRequest) bool {
	args := []string{
		"-c",
		fmt.Sprintf("ssh -o 'StrictHostKeyChecking no' %s@%s -p %d 'echo 1'", req.SSHUser, req.SSHHost, req.SSHPort),
	}

	outs, err := b.cmd.RunCommand(bash, args)
	if err != nil {
		b.log.Error("ssh.tunnel[key].error.outs[%+v:%v]", err, outs)
		return false
	}
	return true
}

func (b *Backup) backupCommands(iskey bool, req *model.BackupRPCRequest) []string {
	var arg string
	var backup string
	var ssh string

	if b.conf.Passwd == "" {
		backup = fmt.Sprintf("%s/xtrabackup --defaults-file=%s --host=%s --port=%d --user=%s --backup --throttle=%d --parallel=%d --stream=xbstream --target-dir=./",
			b.conf.XtrabackupBinDir,
			b.conf.DefaultsFile,
			b.conf.Host,
			b.conf.Port,
			b.conf.Admin,
			req.IOPSLimits,
			b.conf.Parallel)
	} else {
		backup = fmt.Sprintf("%s/xtrabackup --defaults-file=%s --host=%s --port=%d --user=%s --password=%s --backup --throttle=%d --parallel=%d --stream=xbstream --target-dir=./",
			b.conf.XtrabackupBinDir,
			b.conf.DefaultsFile,
			b.conf.Host,
			b.conf.Port,
			b.conf.Admin,
			b.conf.Passwd,
			req.IOPSLimits,
			b.conf.Parallel)
	}

	if iskey {
		ssh = fmt.Sprintf("ssh -o 'StrictHostKeyChecking=no' %s@%s -p %d \"%s/xbstream -x -C %s\"",
			req.SSHUser,
			req.SSHHost,
			req.SSHPort,
			req.XtrabackupBinDir,
			req.BackupDir)
	} else {
		ssh = fmt.Sprintf("sshpass -p %s ssh -o 'StrictHostKeyChecking=no' %s@%s -p %d \"%s/xbstream -x -C %s\"",
			req.SSHPasswd,
			req.SSHUser,
			req.SSHHost,
			req.SSHPort,
			req.XtrabackupBinDir,
			req.BackupDir)
	}
	arg = fmt.Sprintf("%s | %s", backup, ssh)
	b.log.Warning(arg)
	return []string{
		"-c",
		arg,
	}
}

// Backup used to start a backup job.
// If we got CHECKTIMES BACKUPOK in outputs, the backup is completed.
func (b *Backup) Backup(req *model.BackupRPCRequest) error {
	log := b.log

	log.Info("backup.prepare.to.run")
	if b.getStatus() == model.MYSQLD_BACKUPING {
		return errors.New("do.backup.error[backup.job.is.already.running]")
	}

	// check ssh tunnel
	var sshPasswdOK, sshKeyOK bool
	b.log.Info("backup.prepare.to.check.ssh.tunnel")
	sshPasswdOK = b.checkSSHTunnelWithPass(req)
	if !sshPasswdOK {
		b.log.Error("backup.ssh.tunnel[password].error")
		sshKeyOK = b.checkSSHTunnelWithKey(req)
		if !sshKeyOK {
			log.Error("backup.ssh.tunnel[key].error")
			b.setLastError("backup.ssh.tunnel[key].error")
			return fmt.Errorf("backup.ssh.tunnel.to[%v@%v port:%v passwd:%v].can.not.connect", req.SSHUser, req.SSHHost, req.SSHPort, req.SSHPasswd)
		}
	}
	log.Warning("backup.check.ssh[%v].tunnel.done", sshKeyOK)

	b.start = time.Now()
	b.setStatus(model.MYSQLD_BACKUPING)

	args := b.backupCommands(sshKeyOK, req)
	b.setLastCMD(strings.Join(args, " "))
	log.Warning("backup.cmd[%s]", b.getLastCMD())
	if err := b.cmd.Run(bash, args); err != nil {
		b.setLastError(err.Error())
		b.setStatus(model.MYSQLD_BACKUPNONE)
		b.IncBackupErrs()
		log.Error("backup.cmd.run.error[%+v]", err)
		return err
	}

	if err := b.cmd.Scan(backupOk, backupOkCheckTimes); err != nil {
		b.setLastError(err.Error())
		b.setStatus(model.MYSQLD_BACKUPNONE)
		b.IncBackupErrs()
		log.Error("backup.cmd.scan.error[%+v]", err)
		return err
	}

	b.setStatus(model.MYSQLD_BACKUPNONE)
	b.IncBackups()
	log.Warning("backup.done")
	return nil
}

// Cancel used to cancel a backup/applylog job.
func (b *Backup) Cancel() error {
	b.log.Warning("backup.cmd.cancel...")
	b.setStatus(model.MYSQLD_BACKUPCANCELED)
	b.IncCancels()
	return b.cmd.Kill()
}

func (b *Backup) applylogCommands(req *model.BackupRPCRequest) []string {
	arg := fmt.Sprintf("%s/xtrabackup --defaults-file=%s --use-memory=%s --prepare --target-dir=%s", b.conf.XtrabackupBinDir, b.conf.DefaultsFile, b.conf.UseMemory, req.BackupDir)
	return []string{
		"-c",
		arg,
	}
}

// ApplyLog used to apply log from backupdir.
func (b *Backup) ApplyLog(req *model.BackupRPCRequest) error {
	log := b.log

	log.Info("applylog.prepare.to.run")
	if b.getStatus() == model.MYSQLD_BACKUPING ||
		b.getStatus() == model.MYSQLD_APPLYLOGGING {
		return errors.New("applylog.error[backup/applylog.already.running]")
	}

	b.setStatus(model.MYSQLD_APPLYLOGGING)

	args := b.applylogCommands(req)
	log.Warning("applylog.cmd[%s]", strings.Join(args, " "))
	if err := b.cmd.Run(bash, args); err != nil {
		b.setLastError(err.Error())
		log.Error("applylog.cmd.run.error[%+v]", err)
		b.setStatus(model.MYSQLD_BACKUPNONE)
		b.IncApplyLogErrs()
		return err
	}

	if err := b.cmd.Scan(backupOk, backupOkCheckTimes); err != nil {
		b.setLastError(err.Error())
		log.Error("applylog.cmd.scan.error[%+v]", err)
		b.setStatus(model.MYSQLD_BACKUPNONE)
		b.IncApplyLogErrs()
		return err
	}

	b.setStatus(model.MYSQLD_BACKUPNONE)
	b.IncApplyLogs()
	log.Warning("applylog.done")
	return nil
}

func (b *Backup) setStatus(s model.MYSQLD_STATUS) {
	b.status = s
}

func (b *Backup) getStatus() model.MYSQLD_STATUS {
	return b.status
}

func (b *Backup) setLastError(e string) {
	b.stats.LastError = e
}

func (b *Backup) getLastError() string {
	return b.stats.LastError
}

func (b *Backup) setLastCMD(m string) {
	b.stats.LastCMD = m
}

func (b *Backup) getLastCMD() string {
	return b.stats.LastCMD
}

func (b *Backup) getBackupStart() time.Time {
	return b.start
}
