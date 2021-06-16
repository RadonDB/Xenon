/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package mysql

import (
	"database/sql"
	"fmt"
	"model"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// PingStart used to start the ping.
func (m *Mysql) PingStart() {
	go func() {
		for range m.pingTicker.C {
			m.Ping()
		}
	}()
	m.log.Info("mysql[%v].startping...", m.getConnStr())
}

// PingStop used to stop the ping.
func (m *Mysql) PingStop() {
	m.pingTicker.Stop()
}

// Promotable used to check whether we can promote to candidate.
// Promotable:
// 1. MySQL is MysqlAlive
// 2. Slave_SQL_Running
//
// NOTES:
// we do not consider Slave_IO_Running to Promotable, because the MySQL of leader maybe down
// the slaves Slave_IO_Running is false, because it's in connecting state
func (m *Mysql) Promotable() bool {
	log := m.log
	promotable := (m.GetState() == model.MysqlAlive)
	if promotable {
		gtid, err := m.GetGTID()
		if err != nil {
			log.Error("can't.promotable.GetGTID.error:%v", err)
			return false
		}
		//Slave_IO_Running = Yes, So master is alive
		//if (gtid.Slave_IO_Running) {
		//	log.Warning("Maybe Master is alive. Slave IO_thread not timeout (slave_net_timeout)")
		//    return false
		//}

		promotable = (gtid.Slave_SQL_Running)
		log.Warning("mysql[%v].Promotable.sql_thread[%v]", m.getConnStr(), promotable)
		if !promotable {
			log.Error("can't.promotable.GetGTID[%+v]", gtid)
		}
	}
	return promotable
}

// SetReadOnly used to set the mysql to readonly.
func (m *Mysql) SetReadOnly() (err error) {
	var db *sql.DB

	if db, err = m.getDB(); err != nil {
		return
	}

	if err = m.mysqlHandler.SetReadOnly(db, true); err != nil {
		return
	}
	m.setOption(MysqlReadonly)
	return
}

// SetReadWrite used to set the mysql to write.
func (m *Mysql) SetReadWrite() (err error) {
	var db *sql.DB

	if db, err = m.getDB(); err != nil {
		return
	}

	if err = m.mysqlHandler.SetReadOnly(db, false); err != nil {
		return
	}
	m.setOption(MysqlReadwrite)
	return
}

// GTIDGreaterThan used to compare the master_log_file and read_master_log_pos between from and this.
func (m *Mysql) GTIDGreaterThan(gtid *model.GTID) (bool, model.GTID, error) {
	log := m.log
	this, err := m.GetGTID()
	if err != nil {
		return false, this, err
	}

	a := strings.ToUpper(fmt.Sprintf("%s:%016d", this.Master_Log_File, this.Read_Master_Log_Pos))
	b := strings.ToUpper(fmt.Sprintf("%s:%016d", gtid.Master_Log_File, gtid.Read_Master_Log_Pos))
	log.Warning("mysql.gtid.compare.this[%v].from[%v]", this, gtid)
	cmp := strings.Compare(a, b)
	// compare seconds behind master
	if cmp == 0 {
		thislag, err1 := strconv.Atoi(this.Seconds_Behind_Master)
		gtidlag, err2 := strconv.Atoi(gtid.Seconds_Behind_Master)
		if err1 == nil && err2 == nil {
			return (thislag < gtidlag), this, nil
		}
	}
	return cmp > 0, this, nil
}

func (m *Mysql) GetLocalGTID(gtid string) (string, error) {
	log := m.log
	if gtid == "" {
		return "", nil
	}

	uuid, err := m.GetUUID()
	if err != nil {
		log.Error("mysql.GetLocalGTID.error[%v]", err)
		return "", err
	}

	s_gtid := strings.Split(gtid, ",")
	for _, gtid := range s_gtid {
		if strings.Contains(gtid, uuid) {
			return gtid, nil
		}
	}

	return "", nil
}

// CheckGTID use to compare the followerGTID and candidateGTID
func (m *Mysql) CheckGTID(followerGTID *model.GTID, candidateGTID *model.GTID) bool {
	log := m.log
	fExecutedGTID := followerGTID.Executed_GTID_Set
	fGTID, err := m.GetLocalGTID(fExecutedGTID)
	if err != nil {
		log.Error("mysql.CheckGTID.error[%v]", err)
	}

	cExecutedGTID := candidateGTID.Executed_GTID_Set
	cGTID, err := m.GetLocalGTID(cExecutedGTID)
	if err != nil {
		log.Error("mysql.CheckGTID.error[%v]", err)
	}

	// follower never generate events, should vote, but if some one execute reset master, this may be error
	// if a normal restart the follower retrived_gtid_set will be "" can't setState(INVALID)
	if fGTID == "" {
		return false
	}

	// candidate has none RetrivedGTID, may be none retrived_gtid_set
	// this means the candidate or new leader has not written, shouldnt vote
	if cGTID == "" {
		return false
	}

	// gtid_sub is not none, means the follower gtid is bigger than candidate gtid
	// if viewdiff<=0 and gtid_sub is not null it must be localcommitted
	gtid_sub, err := m.GetGTIDSubtract(fGTID, cGTID)
	if err != nil {
		log.Error("mysql.CheckGTID.error[%v]", err)
		return false
	} else if err == nil && gtid_sub != "" {
		log.Warning("follower.gtid[%v].bigger.than.remote[%v]", followerGTID, candidateGTID)
		return true
	}
	return false
}

func (m *Mysql) GetGTIDSubtract(subsetGTID string, setGTID string) (string, error) {
	db, err := m.getDB()
	if err != nil {
		return "", err
	}
	return m.mysqlHandler.GetGTIDSubtract(db, subsetGTID, setGTID)
}

// StartSlaveIOThread used to start the slave io thread.
func (m *Mysql) StartSlaveIOThread() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.StartSlaveIOThread(db)
}

// StopSlaveIOThread used to stop the slave io thread.
func (m *Mysql) StopSlaveIOThread() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.StopSlaveIOThread(db)
}

// StartSlave used to start the slave.
func (m *Mysql) StartSlave() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.StartSlave(db)
}

// StopSlave used to stop the slave.
func (m *Mysql) StopSlave() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.StopSlave(db)
}

// ChangeMasterTo used to do the 'change master to' command.
func (m *Mysql) ChangeMasterTo(repl *model.Repl) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.ChangeMasterTo(db, repl)
}

// ChangeToMaster used to do the 'reset slave all' command.
func (m *Mysql) ChangeToMaster() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.ChangeToMaster(db)
}

// ResetSlaveAll used to reset slave.
func (m *Mysql) ResetSlaveAll() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.ResetSlaveAll(db)
}

// WaitUntilAfterGTID used to do 'SELECT WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS' command.
func (m *Mysql) WaitUntilAfterGTID(targetGTID string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.WaitUntilAfterGTID(db, targetGTID)
}

// GetState returns the mysql state.
func (m *Mysql) GetState() model.MysqlState {
	return m.getState()
}

// SetState set the mysql state.
func (m *Mysql) SetState(state model.MysqlState) {
	m.setState(state)
}

// GetOption returns the mysql option.
func (m *Mysql) GetOption() Option {
	return m.getOption()
}

// GetGTID returns the mysql master_binlog and read_master_log_pos.
// 1. first try GetSlaveGTID
// 2. if STEP1) fails, try GetMasterGTID
func (m *Mysql) GetGTID() (model.GTID, error) {
	log := m.log
	gtid := model.GTID{}
	gotGTID, err := m.GetSlaveGTID()
	if err != nil {
		m.log.Error("mysql.get.slave.gtid.error[%v]", err)
		return gtid, err
	}
	log.Info("mysql.slave.status:%v", gotGTID)

	// we are not slave(maybe a former master)
	// try to get master binary log status
	if gotGTID.Slave_IO_Running_Str == "" && gotGTID.Slave_SQL_Running_Str == "" {
		gotGTID, err = m.GetMasterGTID()
		if err != nil {
			m.log.Error("mysql.get.master.gtid.error[%v]", err)
			return gtid, err
		}
		log.Info("mysql.master.status:%v", gotGTID)
	}
	gtid = *gotGTID
	return gtid, nil
}

// GetRepl returns the repl info.
func (m *Mysql) GetRepl() model.Repl {
	return model.Repl{
		Master_Host:   m.conf.ReplHost,
		Master_Port:   m.conf.Port,
		Repl_User:     m.conf.ReplUser,
		Repl_Password: m.conf.ReplPasswd,
	}
}

// RelayMasterLogFile returns RelayMasterLogFile.
func (m *Mysql) RelayMasterLogFile() string {
	return m.pingEntry.Relay_Master_Log_File
}

// WaitMysqlWorks used to wait for the mysqld to work.
func (m *Mysql) WaitMysqlWorks(timeout int) error {
	maxRunTime := time.Duration(timeout) * time.Millisecond
	errChannel := make(chan error, 1)
	go func() {
		for {
			m.Ping()
			if m.GetState() == model.MysqlAlive {
				errChannel <- nil
				break
			}
			time.Sleep(time.Second)
		}
	}()

	select {
	case <-time.After(maxRunTime):
		return errors.Errorf("WaitMysqlWorks.Timeout[%v]", maxRunTime)
	case err := <-errChannel:
		return err
	}
}

// SetGlobalSysVar used to set global variables.
func (m *Mysql) SetGlobalSysVar(varsql string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.SetGlobalSysVar(db, varsql)
}

// SetMasterGlobalSysVar used to set master global variables.
func (m *Mysql) SetMasterGlobalSysVar() error {
	var err error
	log := m.log

	if m.conf.MasterSysVars == "" {
		return nil
	}
	vars := strings.Split(m.conf.MasterSysVars, ";")
	for _, v := range vars {
		setVar := fmt.Sprintf("SET GLOBAL %s", v)
		if e := m.SetGlobalSysVar(setVar); e != nil {
			err = e
			log.Error("mysql[%v].SetMasterGlobalSysVar.error[%v].var[%v]", m.getConnStr(), err, setVar)
		}
	}
	log.Warning("mysql[%v].SetMasterGlobalSysVar[%v]", m.getConnStr(), m.conf.MasterSysVars)
	return err
}

// SetSlaveGlobalSysVar used to set slave global variables.
func (m *Mysql) SetSlaveGlobalSysVar() error {
	var err error
	log := m.log

	if m.conf.SlaveSysVars == "" {
		return nil
	}
	vars := strings.Split(m.conf.SlaveSysVars, ";")
	for _, v := range vars {
		setVar := fmt.Sprintf("SET GLOBAL %s", v)
		if e := m.SetGlobalSysVar(setVar); e != nil {
			err = e
			log.Error("mysql[%v].SetSlaveGlobalSysVar.error[%v].var[%v]", m.getConnStr(), err, setVar)
		}
	}
	log.Warning("mysql[%v].SetSlaveGlobalSysVar[%v]", m.getConnStr(), m.conf.SlaveSysVars)
	return err
}

// ResetMaster used to reset master.
func (m *Mysql) ResetMaster() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.ResetMaster(db)
}

// PurgeBinlogsTo used to purge binlog.
func (m *Mysql) PurgeBinlogsTo(binlog string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.PurgeBinlogsTo(db, binlog)
}

// EnableSemiSyncMaster used to enable the semi-sync on master.
func (m *Mysql) EnableSemiSyncMaster() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.EnableSemiSyncMaster(db)
}

// SetSemiWaitSlaveCount used to set rpl_semi_sync_master_wait_for_slave_count
func (m *Mysql) SetSemiWaitSlaveCount(count int) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.SetSemiWaitSlaveCount(db, count)
}

// DisableSemiSyncMaster used to disable the semi-sync from master.
func (m *Mysql) DisableSemiSyncMaster() error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.DisableSemiSyncMaster(db)
}

// SetSemiSyncMasterTimeout used to set semi-sync master timeout.
func (m *Mysql) SetSemiSyncMasterTimeout(timeout uint64) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.SetSemiSyncMasterTimeout(db, timeout)
}

// CheckUserExists used to check the user exists or not.
func (m *Mysql) CheckUserExists(user string, host string) (bool, error) {
	db, err := m.getDB()
	if err != nil {
		return false, err
	}
	return m.mysqlHandler.CheckUserExists(db, user, host)
}

// GetUser used to get the mysql user list.
func (m *Mysql) GetUser() ([]model.MysqlUser, error) {
	db, err := m.getDB()
	if err != nil {
		return nil, err
	}
	return m.mysqlHandler.GetUser(db)
}

// CreateUser used to create the new user.
func (m *Mysql) CreateUser(user string, host string, passwd string, ssltype string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.CreateUser(db, user, host, passwd, ssltype)
}

// DropUser used to drop a user.
func (m *Mysql) DropUser(user string, host string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.DropUser(db, user, host)
}

// ChangeUserPasswd used to change the user's password.
func (m *Mysql) ChangeUserPasswd(user string, host string, passwd string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.ChangeUserPasswd(db, user, host, passwd)
}

// CreateReplUserWithoutBinlog used to create a repl user without binlog.
func (m *Mysql) CreateReplUserWithoutBinlog(user string, passwd string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.CreateReplUserWithoutBinlog(db, user, passwd)
}

// GrantNormalPrivileges used grant normal privs.
func (m *Mysql) GrantNormalPrivileges(user string, host string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.GrantNormalPrivileges(db, user, host)
}

// CreateUserWithPrivileges used to create a new user with grants.
func (m *Mysql) CreateUserWithPrivileges(user, passwd, database, table, host, privs string, ssl string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.CreateUserWithPrivileges(db, user, passwd, database, table, host, privs, ssl)
}

// GrantReplicationPrivileges used to grant replication privs.
func (m *Mysql) GrantReplicationPrivileges(user string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.GrantReplicationPrivileges(db, user)
}

// GrantAllPrivileges used to grants all privs for the user.
func (m *Mysql) GrantAllPrivileges(user string, host string, passwd string, ssl string) error {
	db, err := m.getDB()
	if err != nil {
		return err
	}
	return m.mysqlHandler.GrantAllPrivileges(db, user, host, passwd, ssl)
}
