/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package mysql

import (
	"config"
	"database/sql"
	"fmt"
	"model"
	"testing"
	"xbase/xlog"
	"xbase/xrpc"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	QueryTimeout    int    = 10000
	SemiSyncTimeout uint64 = 10000
)

// MockGTID tuple.
type MockGTID struct {
	SetQueryTimeoutFn          func(int)
	PingFn                     func(*sql.DB) (*PingEntry, error)
	SetReadOnlyFn              func(*sql.DB, bool) error
	GetMasterGTIDFn            func(*sql.DB) (*model.GTID, error)
	GetSlaveGTIDFn             func(*sql.DB) (*model.GTID, error)
	StartSlaveIOThreadFn       func(*sql.DB) error
	StopSlaveIOThreadFn        func(*sql.DB) error
	StartSlaveFn               func(*sql.DB) error
	StopSlaveFn                func(*sql.DB) error
	ChangeMasterToFn           func(*sql.DB, *model.Repl) error
	ChangeToMasterFn           func(*sql.DB) error
	WaitUntilAfterGTIDFn       func(*sql.DB, string) error
	GetGTIDSubtractFn          func(*sql.DB, string, string) (string, error)
	GetUUIDFn                  func(*sql.DB) (string, error)
	CheckGTIDFn                func(*model.GTID, *model.GTID) bool
	SetGlobalSysVarFn          func(*sql.DB, string) error
	ResetMasterFn              func(*sql.DB) error
	ResetSlaveAllFn            func(*sql.DB) error
	PurgeBinlogsToFn           func(*sql.DB, string) error
	EnableSemiSyncMasterFn     func(*sql.DB) error
	DisableSemiSyncMasterFn    func(*sql.DB) error
	SelectSysVarFn             func(*sql.DB, string) (string, error)
	SetSemiWaitSlaveCountFn    func(*sql.DB, int) error
	SetSemiSyncMasterTimeoutFn func(*sql.DB, uint64) error

	// Users
	GetUserFn                     func(*sql.DB) ([]model.MysqlUser, error)
	CheckUserExistsFn             func(*sql.DB, string, string) (bool, error)
	CreateUserFn                  func(*sql.DB, string, string, string, string) error
	DropUserFn                    func(*sql.DB, string, string) error
	ChangeUserPasswdFn            func(*sql.DB, string, string, string) error
	CreateReplUserWithoutBinlogFn func(*sql.DB, string, string) error
	GrantAllPrivilegesFn          func(*sql.DB, string, string, string, string) error
	GrantNormalPrivilegesFn       func(*sql.DB, string, string) error
	CreateUserWithPrivilegesFn    func(*sql.DB, string, string, string, string, string, string, string) error
	GrantReplicationPrivilegesFn  func(*sql.DB, string) error
}

// DefaultGetSlaveGTID returns the default slave gtid.
func DefaultGetSlaveGTID(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	return gtid, nil
}

// GetSlaveGTID mock.
func (mogtid *MockGTID) GetSlaveGTID(db *sql.DB) (*model.GTID, error) {
	return mogtid.GetSlaveGTIDFn(db)
}

// DefaultGetUUID mock.
func DefaultGetUUID(db *sql.DB) (string, error) {
	return "84030605-66aa-11e6-9465-52540e7fd51c", nil
}

// GetUUID mock.
func (mogtid *MockGTID) GetUUID(db *sql.DB) (string, error) {
	return mogtid.GetUUIDFn(db)
}

// DefaultGetMasterGTID mock.
func DefaultGetMasterGTID(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	return gtid, nil
}

// GetMasterGTID mock.
func (mogtid *MockGTID) GetMasterGTID(db *sql.DB) (*model.GTID, error) {
	return mogtid.GetMasterGTIDFn(db)
}

// DefaultStartSlaveIOThread mock.
func DefaultStartSlaveIOThread(db *sql.DB) error {
	return nil
}

// StartSlaveIOThread mock.
func (mogtid *MockGTID) StartSlaveIOThread(db *sql.DB) error {
	return mogtid.StartSlaveIOThreadFn(db)
}

// DefaultStopSlaveIOThread mock.
func DefaultStopSlaveIOThread(db *sql.DB) error {
	return nil
}

// StopSlaveIOThread mock.
func (mogtid *MockGTID) StopSlaveIOThread(db *sql.DB) error {
	return mogtid.StartSlaveIOThreadFn(db)
}

// DefaultStartSlave mock.
func DefaultStartSlave(db *sql.DB) error {
	return nil
}

// StartSlave mock.
func (mogtid *MockGTID) StartSlave(db *sql.DB) error {
	return mogtid.StartSlaveFn(db)
}

// DefaultStopSlave mock.
func DefaultStopSlave(db *sql.DB) error {
	return nil
}

// StopSlave mock.
func (mogtid *MockGTID) StopSlave(db *sql.DB) error {
	return mogtid.StopSlaveFn(db)
}

// DefaultChangeMasterTo mock.
func DefaultChangeMasterTo(db *sql.DB, gtid *model.Repl) error {
	return nil
}

// ChangeMasterTo mock.
func (mogtid *MockGTID) ChangeMasterTo(db *sql.DB, gtid *model.Repl) error {
	return mogtid.ChangeMasterToFn(db, gtid)
}

// DefaultChangeToMaster mock.
func DefaultChangeToMaster(db *sql.DB) error {
	return nil
}

// ChangeToMaster mock.
func (mogtid *MockGTID) ChangeToMaster(db *sql.DB) error {
	return mogtid.ChangeToMasterFn(db)
}

// DefaultWaitUntilAfterGTID mock.
func DefaultWaitUntilAfterGTID(db *sql.DB, targetGTID string) error {
	return nil
}

// WaitUntilAfterGTID mock.
func (mogtid *MockGTID) WaitUntilAfterGTID(db *sql.DB, targetGTID string) error {
	return mogtid.WaitUntilAfterGTIDFn(db, targetGTID)
}

// DefaultGetGTIDSubtract mock.
func DefaultGetGTIDSubtract(db *sql.DB, slaveGTID string, masterGTID string) (string, error) {
	return "", nil
}

func DefaultCheckGTID(followerGTID *model.GTID, leaderGTID *model.GTID) bool {
	return false
}

// GetGTIDSubtract mock.
func (mogtid *MockGTID) GetGTIDSubtract(db *sql.DB, slaveGTID string, masterGTID string) (string, error) {
	return mogtid.GetGTIDSubtractFn(db, slaveGTID, masterGTID)
}

// DefaultSetQueryTimeout mock.
func DefaultSetQueryTimeout(timeout int) {
}

// SetQueryTimeout mock.
func (mogtid *MockGTID) SetQueryTimeout(timeout int) {
	mogtid.SetQueryTimeoutFn(timeout)
}

// DefaultPing mock.
func DefaultPing(db *sql.DB) (*PingEntry, error) {
	return &PingEntry{}, nil
}

// Ping mock.
func (mogtid *MockGTID) Ping(db *sql.DB) (*PingEntry, error) {
	return mogtid.PingFn(db)
}

// DefaultSetReadOnly mock.
func DefaultSetReadOnly(db *sql.DB, readonly bool) error {
	return nil
}

// SetReadOnly mock.
func (mogtid *MockGTID) SetReadOnly(db *sql.DB, readonly bool) error {
	return mogtid.SetReadOnlyFn(db, readonly)
}

// DefaultSetGlobalSysVar mock.
func DefaultSetGlobalSysVar(db *sql.DB, varsql string) error {
	return nil
}

// SetGlobalSysVar mock.
func (mogtid *MockGTID) SetGlobalSysVar(db *sql.DB, varsql string) error {
	return mogtid.SetGlobalSysVarFn(db, varsql)
}

// DefaultResetMaster mock.
func DefaultResetMaster(db *sql.DB) error {
	return nil
}

// ResetMaster mock.
func (mogtid *MockGTID) ResetMaster(db *sql.DB) error {
	return mogtid.ResetMasterFn(db)
}

// DefaultResetSlaveAll mock.
func DefaultResetSlaveAll(db *sql.DB) error {
	return nil
}

// ResetSlaveAll mock.
func (mogtid *MockGTID) ResetSlaveAll(db *sql.DB) error {
	return mogtid.ResetSlaveAllFn(db)
}

// DefaultPurgeBinlogsTo mock.
func DefaultPurgeBinlogsTo(db *sql.DB, binlog string) error {
	return nil
}

// PurgeBinlogsTo mock.
func (mogtid *MockGTID) PurgeBinlogsTo(db *sql.DB, binlog string) error {
	return mogtid.PurgeBinlogsToFn(db, binlog)
}

// DefaultEnableSemiSyncMaster mock.
func DefaultEnableSemiSyncMaster(db *sql.DB) error {
	return nil
}

// EnableSemiSyncMaster mock.
func (mogtid *MockGTID) EnableSemiSyncMaster(db *sql.DB) error {
	return mogtid.EnableSemiSyncMasterFn(db)
}

// DefaultDisableSemiSyncMaster mock.
func DefaultDisableSemiSyncMaster(db *sql.DB) error {
	return nil
}

// DisableSemiSyncMaster mock.
func (mogtid *MockGTID) DisableSemiSyncMaster(db *sql.DB) error {
	return mogtid.DisableSemiSyncMasterFn(db)
}

// SetSemiSyncMasterTimeout mock.
func (mogtid *MockGTID) SetSemiSyncMasterTimeout(db *sql.DB, timeout uint64) error {
	return mogtid.SetSemiSyncMasterTimeoutFn(db, timeout)
}

// DefaultSelectSysVar mock.
func DefaultSelectSysVar(db *sql.DB, query string) (string, error) {
	return "", nil
}

// SelectSysVar mock.
func (mogtid *MockGTID) SelectSysVar(db *sql.DB, query string) (string, error) {
	return mogtid.SelectSysVarFn(db, query)
}

// DefaultSetSemiWaitSlaveCount mock.
func DefaultSetSemiWaitSlaveCount(db *sql.DB, count int) error {
	return nil
}

// SetSemiWaitSlaveCount mock
func (mogtid *MockGTID) SetSemiWaitSlaveCount(db *sql.DB, count int) error {
	return mogtid.SetSemiWaitSlaveCountFn(db, count)
}

// SetSemiSyncMasterTimeout mock
func SetSemiSyncMasterTimeout(db *sql.DB, timeout uint64) error {
	return nil
}

// User handlers.

// CheckUserExists mock.
func DefaultCheckUserExists(db *sql.DB, user string, host string) (bool, error) {
	return false, nil
}

func (mogtid *MockGTID) CheckUserExists(db *sql.DB, user string, host string) (bool, error) {
	return mogtid.CheckUserExistsFn(db, user, host)
}

// CreateUser mock.
func DefaultCreateUser(db *sql.DB, user string, host string, passwd string, ssltype string) error {
	return nil
}

func (mogtid *MockGTID) CreateUser(db *sql.DB, user string, host string, passwd string, ssltype string) error {
	return mogtid.CreateUserFn(db, user, host, passwd, ssltype)
}

// GetUser mock.
func DefaultGetUser(db *sql.DB) ([]model.MysqlUser, error) {
	return []model.MysqlUser{
		{User: "user1",
			Host:      "localhost",
			SuperPriv: "N"},
		{User: "root",
			Host:      "localhost",
			SuperPriv: "Y"},
	}, nil
}

func (mogtid *MockGTID) GetUser(db *sql.DB) ([]model.MysqlUser, error) {
	return mogtid.GetUserFn(db)
}

// CreateUserWithPrivileges mock.
func DefaultCreateUserWithPrivileges(db *sql.DB, user, passwd, database, table, host, privs string, ssl string) error {
	return nil
}

func (mogtid *MockGTID) CreateUserWithPrivileges(db *sql.DB, user, passwd, database, table, host, privs string, ssl string) error {
	return mogtid.CreateUserWithPrivilegesFn(db, user, passwd, database, table, host, privs, ssl)
}

// DropUser mock.
func DefaultDropUser(db *sql.DB, user string, host string) error {
	return nil
}

func (mogtid *MockGTID) DropUser(db *sql.DB, user string, host string) error {
	return mogtid.DropUserFn(db, user, host)
}

// CreateReplUserWithoutBinlog mock.
func DefaultCreateReplUserWithoutBinlog(db *sql.DB, user string, passwd string) error {
	return nil
}

func (mogtid *MockGTID) CreateReplUserWithoutBinlog(db *sql.DB, user string, passwd string) error {
	return mogtid.CreateReplUserWithoutBinlogFn(db, user, passwd)
}

// ChangeUserPasswd mock.
func DefaultChangeUserPasswd(db *sql.DB, user string, host string, passwd string) error {
	return nil
}

func (mogtid *MockGTID) ChangeUserPasswd(db *sql.DB, user string, host string, passwd string) error {
	return mogtid.ChangeUserPasswdFn(db, user, host, passwd)
}

// GrantNormalPrivileges mock.
func DefaultGrantNormalPrivileges(db *sql.DB, user string, host string) error {
	return nil
}

func (mogtid *MockGTID) GrantNormalPrivileges(db *sql.DB, user string, host string) error {
	return mogtid.GrantNormalPrivilegesFn(db, user, host)
}

// GrantReplicationPrivileges mock.
func DefaultGrantReplicationPrivileges(db *sql.DB, user string) error {
	return nil
}

func (mogtid *MockGTID) GrantReplicationPrivileges(db *sql.DB, user string) error {
	return mogtid.GrantReplicationPrivilegesFn(db, user)
}

// GrantAllPrivileges mock.
func DefaultGrantAllPrivileges(db *sql.DB, user string, host string, passwd string, ssl string) error {
	return nil
}

func (mogtid *MockGTID) GrantAllPrivileges(db *sql.DB, user string, host, passwd, ssl string) error {
	return mogtid.GrantAllPrivilegesFn(db, user, host, passwd, ssl)
}

func defaultMockGTID() *MockGTID {
	mock := &MockGTID{}
	mock.SetQueryTimeoutFn = DefaultSetQueryTimeout
	mock.PingFn = DefaultPing
	mock.SetReadOnlyFn = DefaultSetReadOnly
	mock.GetMasterGTIDFn = DefaultGetMasterGTID
	mock.GetSlaveGTIDFn = DefaultGetSlaveGTID
	mock.StartSlaveIOThreadFn = DefaultStartSlaveIOThread
	mock.StopSlaveIOThreadFn = DefaultStopSlaveIOThread
	mock.StartSlaveFn = DefaultStartSlave
	mock.StopSlaveFn = DefaultStopSlave
	mock.ChangeMasterToFn = DefaultChangeMasterTo
	mock.ChangeToMasterFn = DefaultChangeToMaster
	mock.WaitUntilAfterGTIDFn = DefaultWaitUntilAfterGTID
	mock.GetGTIDSubtractFn = DefaultGetGTIDSubtract
	mock.GetUUIDFn = DefaultGetUUID
	mock.CheckGTIDFn = DefaultCheckGTID
	mock.SetGlobalSysVarFn = DefaultSetGlobalSysVar
	mock.ResetMasterFn = DefaultResetMaster
	mock.ResetSlaveAllFn = DefaultResetSlaveAll
	mock.PurgeBinlogsToFn = DefaultPurgeBinlogsTo
	mock.EnableSemiSyncMasterFn = DefaultEnableSemiSyncMaster
	mock.DisableSemiSyncMasterFn = DefaultDisableSemiSyncMaster
	mock.SelectSysVarFn = DefaultSelectSysVar
	mock.SetSemiWaitSlaveCountFn = DefaultSetSemiWaitSlaveCount
	mock.SetSemiSyncMasterTimeoutFn = SetSemiSyncMasterTimeout

	// Users.
	mock.CheckUserExistsFn = DefaultCheckUserExists
	mock.CreateUserFn = DefaultCreateUser
	mock.GetUserFn = DefaultGetUser
	mock.CreateUserWithPrivilegesFn = DefaultCreateUserWithPrivileges
	mock.DropUserFn = DefaultDropUser
	mock.CreateReplUserWithoutBinlogFn = DefaultCreateReplUserWithoutBinlog
	mock.ChangeUserPasswdFn = DefaultChangeUserPasswd
	mock.GrantNormalPrivilegesFn = DefaultGrantNormalPrivileges
	mock.GrantReplicationPrivilegesFn = DefaultGrantReplicationPrivileges
	mock.GrantAllPrivilegesFn = DefaultGrantAllPrivileges

	return mock
}

// GetSlaveGTIDA mock.
// with GTID{Master_Log_File = "", Read_Master_Log_Pos = 0}
// all functions return is OK
func GetSlaveGTIDA(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = ""
	gtid.Read_Master_Log_Pos = 0
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	gtid.Slave_IO_Running_Str = "Yes"
	gtid.Slave_SQL_Running_Str = "Yes"
	gtid.Seconds_Behind_Master = "1"
	gtid.Last_Error = ""
	gtid.Slave_SQL_Running_State = "Slave has read all relay log; waiting for the slave I/O thread to update it"
	gtid.Executed_GTID_Set = `052077a5-b6f4-ee1b-61ec-d80a8b27d749:1-36,
    12446bf7-3219-11e5-9434-080027079e3d:8058-963126`
	gtid.Retrieved_GTID_Set = `052077a5-b6f4-ee1b-61ec-d80a8b27d749:1-36,
    12446bf7-3219-11e5-9434-080027079e3d:8058-963126`
	return gtid, nil
}

// GetMasterGTIDA mock.
func GetMasterGTIDA(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = ""
	gtid.Read_Master_Log_Pos = 0
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	gtid.Seconds_Behind_Master = "0"
	gtid.Last_Error = ""
	gtid.Slave_SQL_Running_State = ""
	gtid.Executed_GTID_Set = `052077a5-b6f4-ee1b-61ec-d80a8b27d749:1-36,
    12446bf7-3219-11e5-9434-080027079e3d:8058-963126`
	gtid.Retrieved_GTID_Set = `052077a5-b6f4-ee1b-61ec-d80a8b27d749:1-36,
    12446bf7-3219-11e5-9434-080027079e3d:8058-963126`
	return gtid, nil
}

func GetUUIDA(db *sql.DB) (string, error) {
	return "052077a5-b6f4-ee1b-61ec-d80a8b27d749", nil
}

// NewMockGTIDA mock.
func NewMockGTIDA() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDA
	mock.GetSlaveGTIDFn = GetSlaveGTIDA
	mock.GetUUIDFn = GetUUIDA
	return mock
}

func GetUUIDLC(db *sql.DB) (string, error) {
	return "052077a5-b6f4-ee1b-61ec-d80a8b27d749", nil
}

// GetMasterGTIDLC mock.
func GetMasterGTIDLC(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = ""
	gtid.Read_Master_Log_Pos = 0
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	gtid.Seconds_Behind_Master = "0"
	gtid.Last_Error = ""
	gtid.Slave_SQL_Running_State = ""
	gtid.Executed_GTID_Set = `052077a5-b6f4-ee1b-61ec-d80a8b27d749:1-37,
    12446bf7-3219-11e5-9434-080027079e3d:8058-963126`
	gtid.Retrieved_GTID_Set = `052077a5-b6f4-ee1b-61ec-d80a8b27d749:1-37,
    12446bf7-3219-11e5-9434-080027079e3d:8058-963126`
	return gtid, nil
}

func CheckGTIDLC(*model.GTID, *model.GTID) bool {
	return true
}

// NewMockGTIDLC mock
func NewMockGTIDLC() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDLC
	mock.GetSlaveGTIDFn = GetMasterGTIDLC
	mock.CheckGTIDFn = CheckGTIDLC
	mock.GetGTIDSubtractFn = GetGTIDSubtractInvalid
	mock.GetUUIDFn = GetUUIDLC
	return mock
}

// NewMockGTIDAA mock.
// with GTID{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123}
// all functions return is OK
func NewMockGTIDAA() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDAA
	mock.GetSlaveGTIDFn = GetSlaveGTIDAA
	return mock
}

// GetSlaveGTIDB mock.
func GetSlaveGTIDAA(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 122
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1"
	gtid.Retrieved_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1"
	return gtid, nil
}

// GetMasterGTIDB mock.
func GetMasterGTIDAA(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 122
	gtid.Executed_GTID_Set = ""
	gtid.Retrieved_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDB mock.
// with GTID{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123}
// all functions return is OK
func NewMockGTIDB() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDB
	mock.GetSlaveGTIDFn = GetSlaveGTIDB
	return mock
}

// GetSlaveGTIDB mock.
func GetSlaveGTIDB(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 123
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-2"
	gtid.Retrieved_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-2"
	return gtid, nil
}

// GetMasterGTIDB mock.
func GetMasterGTIDB(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 123
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-2"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDBB mock.
// with GTID{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123}
// all functions return is OK
func NewMockGTIDBB() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDBB
	mock.GetSlaveGTIDFn = GetSlaveGTIDBB
	return mock
}

// GetSlaveGTIDBB mock.
func GetSlaveGTIDBB(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 123
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1, df24366e-bbbb-bbbb-bbbb-525433b6dbaa:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// GetMasterGTIDBB mock.
func GetMasterGTIDBB(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 123
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1, df24366e-inva-bbbb-bbbb-525433b6dbaa:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDC mock.
// with GTID{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 124}
// all functions return is OK
func NewMockGTIDC() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDC
	mock.GetSlaveGTIDFn = GetSlaveGTIDC
	return mock
}

// GetSlaveGTIDC mock.
func GetSlaveGTIDC(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 124
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	gtid.Slave_IO_Running_Str = "Yes"
	gtid.Slave_SQL_Running_Str = "Yes"
	gtid.Retrieved_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-3"
	return gtid, nil
}

// GetMasterGTIDC mock.
func GetMasterGTIDC(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 125
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-3"
	gtid.Retrieved_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-3"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDCC mock.
// with GTID{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 124}
// all functions return is OK
func NewMockGTIDCC() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDCC
	mock.GetSlaveGTIDFn = GetSlaveGTIDCC
	return mock
}

// GetSlaveGTIDC mock.
func GetSlaveGTIDCC(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 124
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1, df24366e-inva-bbbb-bbbb-525433b6dbaa:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	gtid.Slave_IO_Running_Str = "Yes"
	gtid.Slave_SQL_Running_Str = "Yes"
	return gtid, nil
}

// GetMasterGTIDC mock.
func GetMasterGTIDCC(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 125
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1, df24366e-inva-bbbb-bbbb-525433b6dbaa:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDD mock.
func NewMockGTIDD() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDD
	return mock
}

// GetMasterGTIDD mock.
func GetMasterGTIDD(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 126
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1, df24366e-inva-bbbb-bbbb-525433b6dbaa:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDE1 mock.
func NewMockGTIDE1() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDE1
	mock.GetGTIDSubtractFn = GetGTIDSubtractE
	return mock
}

// GetMasterGTIDE1 mock.
func GetMasterGTIDE1(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 126
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-200, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-200, ef24366e-aaaa-aaaa-aaaa-525433b6deee:100"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDE2 mock.
func NewMockGTIDE2() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDE2
	mock.GetGTIDSubtractFn = GetGTIDSubtractE
	return mock
}

// GetMasterGTIDE2 mock.
func GetMasterGTIDE2(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 126
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-10, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-40"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDE3 mock.
func NewMockGTIDE3() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDE3
	mock.GetGTIDSubtractFn = GetGTIDSubtractE
	return mock
}

// GetMasterGTIDE3 mock.
func GetMasterGTIDE3(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 126
	gtid.Executed_GTID_Set = "df24366e-inva-bbbb-bbbb-525433b6dbaa:1-31"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// GetGTIDSubtractE mock
func GetGTIDSubtractE(db *sql.DB, subsetGTID string, setGTID string) (string, error) {
	if setGTID != "c78e798a-cccc-cccc-cccc-525433e8e796:1-10, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-30" {
		return "", nil
	}

	switch subsetGTID {
	case "c78e798a-cccc-cccc-cccc-525433e8e796:1-200, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-200, ef24366e-aaaa-aaaa-aaaa-525433b6deee:100":
		return "c78e798a-cccc-cccc-cccc-525433e8e796:11-200,\ndf24366e-inva-bbbb-bbbb-525433b6dbaa:31-200,\nef24366e-aaaa-aaaa-aaaa-525433b6deee:100", nil
	case "c78e798a-cccc-cccc-cccc-525433e8e796:1-10, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-40":
		return "df24366e-inva-bbbb-bbbb-525433b6dbaa:31-40", nil
	case "df24366e-inva-bbbb-bbbb-525433b6dbaa:1-31":
		return "df24366e-inva-bbbb-bbbb-525433b6dbaa:31", nil
	}

	return "", nil
}

// NewMockGTIDF mock.
func NewMockGTIDF() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDF
	mock.GetGTIDSubtractFn = GetGTIDSubtractE
	return mock
}

// GetMasterGTIDF mock.
func GetMasterGTIDF(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 126
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1-10, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-30"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDNull mock.
func NewMockGTIDNull() *MockGTID {
	mock := defaultMockGTID()
	mock.GetMasterGTIDFn = GetMasterGTIDNull
	return mock
}

// GetMasterGTIDNull mock.
func GetMasterGTIDNull(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 126
	gtid.Executed_GTID_Set = ""
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDGetSubtractError mock.
func NewMockGTIDGetGTIDSubtractError() *MockGTID {
	mock := defaultMockGTID()
	mock.GetGTIDSubtractFn = GetGTIDSubtractError
	return mock
}

func GetGTIDSubtractError(db *sql.DB, subsetGTID string, setGTID string) (string, error) {
	return "", errors.New("mock.GetGTIDSubtract.error")
}

// NewMockGTIDPingError mock.
// mock Ping returns error
func NewMockGTIDPingError() *MockGTID {
	mock := defaultMockGTID()
	mock.PingFn = PingError1
	mock.GetMasterGTIDFn = GetMasterGTIDPingError
	return mock
}

// GetMasterGTIDPingError mock.
func GetMasterGTIDPingError(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 124
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// PingError1 mock.
func PingError1(db *sql.DB) (*PingEntry, error) {
	return nil, errors.New("MockGTIDPingError.ping.error")
}

// NewMockGTIDInvalid mock.
// mock GetSlaveGTIDInvalid returns Invalid
// mock GetMasterGTIDInvalid returns Invalid
func NewMockGTIDInvalid() *MockGTID {
	mock := defaultMockGTID()
	mock.GetGTIDSubtractFn = GetGTIDSubtractInvalid
	mock.GetMasterGTIDFn = GetMasterGTIDInvalid
	return mock
}

func GetMasterGTIDInvalid(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 122
	gtid.Executed_GTID_Set = "c78e798a-cccc-cccc-cccc-525433e8e796:1, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-3"
	gtid.Slave_IO_Running = true
	gtid.Slave_SQL_Running = true
	return gtid, nil
}

// NewMockGTIDError mock.
// mock GetSlaveGTID returns error
// mock GetMasterGTID returns error
func NewMockGTIDError() *MockGTID {
	mock := defaultMockGTID()
	mock.PingFn = PingError2
	mock.SetReadOnlyFn = SetReadOnlyError
	mock.GetMasterGTIDFn = GetMasterGTIDError
	mock.GetSlaveGTIDFn = GetSlaveGTIDError
	mock.StartSlaveIOThreadFn = StartSlaveIOThreadError
	mock.StopSlaveIOThreadFn = StopSlaveIOThreadError
	mock.StartSlaveFn = StartSlaveError
	mock.StopSlaveFn = StopSlaveError
	mock.ChangeMasterToFn = ChangeMasterToError
	mock.ChangeToMasterFn = ChangeToMasterError
	mock.WaitUntilAfterGTIDFn = WaitUntilAfterGTIDError
	return mock
}

// GetSlaveGTIDError mock.
func GetSlaveGTIDError(db *sql.DB) (*model.GTID, error) {
	return nil, errors.New("mock.GetSlaveGTID.error")
}

// GetMasterGTIDError mock.
func GetMasterGTIDError(db *sql.DB) (*model.GTID, error) {
	return nil, errors.New("mock.GetMasterGTID.error")
}

// StartSlaveIOThreadError mock.
func StartSlaveIOThreadError(db *sql.DB) error {
	return errors.New("mock.StartSlaveIOThread.error")
}

// StopSlaveError mock.
func StopSlaveError(db *sql.DB) error {
	return errors.New("mock.StopSlave.error")
}

// StartSlaveError mock.
func StartSlaveError(db *sql.DB) error {
	return errors.New("mock.StartSlave.error")
}

// StopSlaveIOThreadError mock.
func StopSlaveIOThreadError(db *sql.DB) error {
	return errors.New("mock.StopSlaveIOThread.error")
}

// ChangeMasterToError mock.
func ChangeMasterToError(db *sql.DB, gtid *model.Repl) error {
	return errors.New("mock.ChangeMasterTo.error")
}

// ChangeToMasterError mock.
func ChangeToMasterError(db *sql.DB) error {
	return errors.New("mock.ChangeMasterTo.error")
}

// WaitUntilAfterGTIDError mock.
func WaitUntilAfterGTIDError(db *sql.DB, targetGTID string) error {
	return errors.New("mock.WaitUntilAfterGTID.error")
}

// GetGTIDSubtractInvalid mock.
func GetGTIDSubtractInvalid(db *sql.DB, slaveGTID string, masterGTID string) (string, error) {
	return "localcommit", nil
}

// PingError2 mock.
func PingError2(db *sql.DB) (*PingEntry, error) {
	return nil, errors.New("MockGTIDE.ping.error")
}

// SetReadOnlyError mock.
func SetReadOnlyError(db *sql.DB, readonly bool) error {
	return errors.New("mock.SetReadOnly.error")
}

func setupRPC(rpc *xrpc.Service, mysql *Mysql) {
	if err := rpc.RegisterService(mysql.GetMysqlRPC()); err != nil {
		mysql.log.Panic("server.rpc.RegisterService.MysqlRPC.error[%v]", err)
	}
}

// NewMockGTIDX1 mock.
// with GTID{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123,
//           gtid.Executed_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1"}
// all functions return is OK
func NewMockGTIDX1() *MockGTID {
	mock := defaultMockGTID()
	mock.PingFn = PingX1
	mock.GetSlaveGTIDFn = GetSlaveGTIDX1
	return mock
}

// PingX1 mock.
func PingX1(db *sql.DB) (*PingEntry, error) {
	return &PingEntry{"mysql-bin.000001"}, nil
}

// GetSlaveGTIDX1 mock.
func GetSlaveGTIDX1(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000001"
	gtid.Read_Master_Log_Pos = 123
	gtid.Retrieved_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1"
	gtid.Executed_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_IO_Running_Str = "Yes"
	gtid.Slave_SQL_Running = true
	gtid.Slave_IO_Running_Str = "Yes"
	return gtid, nil
}

// NewMockGTIDX3 mock.
// with GTID{Master_Log_File = "mysql-bin.000003", Read_Master_Log_Pos = 123 ,
//           gtid.Executed_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1"}
// all functions return is OK
func NewMockGTIDX3() *MockGTID {
	mock := defaultMockGTID()
	mock.PingFn = PingX3
	mock.GetSlaveGTIDFn = GetSlaveGTIDX3
	return mock
}

// PingX3 mock.
func PingX3(db *sql.DB) (*PingEntry, error) {
	return &PingEntry{"mysql-bin.000003"}, nil
}

// GetSlaveGTIDX3 mock.
func GetSlaveGTIDX3(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000003"
	gtid.Read_Master_Log_Pos = 123
	gtid.Retrieved_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1-2"
	gtid.Executed_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1"
	gtid.Slave_IO_Running = true
	gtid.Slave_IO_Running_Str = "Yes"
	gtid.Slave_SQL_Running = true
	gtid.Slave_SQL_Running_Str = "Yes"
	return gtid, nil
}

// NewMockGTIDX5 mock.
// with GTID{Master_Log_File = "mysql-bin.000005", Read_Master_Log_Pos = 123,
//           gtid.Executed_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1"}
// all functions return is OK
func NewMockGTIDX5() *MockGTID {
	mock := defaultMockGTID()
	mock.PingFn = PingX5
	mock.GetSlaveGTIDFn = GetSlaveGTIDX5
	return mock
}

// PingX5 mock.
func PingX5(db *sql.DB) (*PingEntry, error) {
	return &PingEntry{"mysql-bin.000005"}, nil
}

// GetSlaveGTIDX5 mock.
func GetSlaveGTIDX5(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}
	gtid.Master_Log_File = "mysql-bin.000005"
	gtid.Read_Master_Log_Pos = 123
	gtid.Executed_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1"
	gtid.Retrieved_GTID_Set = "6127a668-gtid-x555-a28d-5254335479b2:1-3"
	gtid.Slave_IO_Running = true
	gtid.Slave_IO_Running_Str = "Yes"
	gtid.Slave_SQL_Running = true
	gtid.Slave_SQL_Running_Str = "Yes"
	return gtid, nil
}

// NewMockGTIDX5WaitUntilAfterGTIDError mock.
// with GTID{Master_Log_File = "mysql-bin.000005", Read_Master_Log_Pos = 123}
// all functions return is OK except WaitUntilAfterGTID
func NewMockGTIDX5WaitUntilAfterGTIDError() *MockGTID {
	mock := defaultMockGTID()
	mock.GetSlaveGTIDFn = GetSlaveGTIDX5
	mock.WaitUntilAfterGTIDFn = WaitUntilAfterGTIDError
	return mock
}

// NewMockGTIDX5ChangeToMasterError mock.
// with GTID{Master_Log_File = "mysql-bin.000005", Read_Master_Log_Pos = 123}
// all functions return is OK except ChangeToMaster error.
func NewMockGTIDX5ChangeToMasterError() *MockGTID {
	mock := defaultMockGTID()
	mock.GetSlaveGTIDFn = GetSlaveGTIDX5
	mock.ChangeToMasterFn = ChangeToMasterError
	return mock
}

// MockMysql mock.
func MockMysql(log *xlog.Log, port int, h MysqlHandler) (string, *Mysql, func()) {
	id := fmt.Sprintf("127.0.0.1:%d", port)
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, QueryTimeout, log)

	// setup rpc
	rpc, err := xrpc.NewService(xrpc.Log(log),
		xrpc.ConnectionStr(id))
	if err != nil {
		log.Panic("mysqlRPC.NewService.error[%v]", err)
	}
	setupRPC(rpc, mysql)
	rpc.Start()

	// Set mock functions
	mysql.SetMysqlHandler(h)

	// start ping
	mysql.PingStart()
	return id, mysql, func() {
		mysql.PingStop()
		rpc.Stop()
	}
}

// MockMysqlReplUser mock.
func MockMysqlReplUser(log *xlog.Log, port int, h MysqlHandler) (string, *Mysql, func()) {
	id := fmt.Sprintf("127.0.0.1:%d", port)
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, QueryTimeout, log)

	// setup rpc
	rpc, err := xrpc.NewService(xrpc.Log(log),
		xrpc.ConnectionStr(id))
	if err != nil {
		log.Panic("mysqlRPC.NewService.error[%v]", err)
	}
	setupRPC(rpc, mysql)
	rpc.Start()

	// Set Repl functions
	mysql.SetMysqlHandler(h)

	// start ping
	mysql.PingStart()
	return id, mysql, func() {
		mysql.PingStop()
		rpc.Stop()
	}
}

// MockGetClient mock.
func MockGetClient(t *testing.T, svrConn string) (*xrpc.Client, func()) {
	client, err := xrpc.NewClient(svrConn, 100)
	assert.Nil(t, err)
	return client, func() {
		client.Close()
	}
}
