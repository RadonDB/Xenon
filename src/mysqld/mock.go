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
	"testing"
	"xbase/common"
	"xbase/xlog"
	"xbase/xrpc"

	"github.com/stretchr/testify/assert"
)

var (
	_ ArgsHandler = &MockArgs{}
)

// mock mysqld with rpc server
func setupRPC(rpc *xrpc.Service, mysqld *Mysqld) {
	if err := rpc.RegisterService(mysqld.GetBackupRPC()); err != nil {
		mysqld.log.Panic("server.rpc.RegisterService.GetBackupRPC.error[%v]", err)
	}
	if err := rpc.RegisterService(mysqld.GetMysqldRPC()); err != nil {
		mysqld.log.Panic("server.rpc.RegisterService.GetMysqldRPC.error[%v]", err)
	}
}

// MockMysqld used to mock a mysqld.
func MockMysqld(log *xlog.Log, port int) (string, *Mysqld, func()) {
	id := fmt.Sprintf("127.0.0.1:%d", port)
	conf := config.DefaultBackupConfig()
	mysqld := NewMysqld(conf, log)
	mysqld.SetArgsHandler(NewMockArgs())
	mysqld.backup.SetCMDHandler(common.NewMockCommand())

	// setup rpc
	rpc, err := xrpc.NewService(xrpc.Log(log),
		xrpc.ConnectionStr(id))
	if err != nil {
		log.Panic("mysqldRPC.NewService.error[%v]", err)
	}
	setupRPC(rpc, mysqld)
	rpc.Start()

	return id, mysqld, func() {
		rpc.Stop()
	}
}

// MockGetClient used to mock client.
func MockGetClient(t *testing.T, svrConn string) (*xrpc.Client, func()) {
	client, err := xrpc.NewClient(svrConn, 100)
	assert.Nil(t, err)

	return client, func() {
		client.Close()
	}
}

// MockArgs tuple.
type MockArgs struct {
	ArgsHandler
}

// NewMockArgs creates the new MockArgs.
func NewMockArgs() *MockArgs {
	return &MockArgs{}
}

// Start used to start the mock.
func (l *MockArgs) GenerateStartCmd() []string {
	args := []string{
		"-c",
		"ls -l",
	}

	return args
}

// Stop used to stop the mock.
func (l *MockArgs) GenerateStopCmd() []string {
	args := []string{
		"-c",
		"ls -l",
	}

	return args
}

// IsRunning used to check the mock running.
func (l *MockArgs) GenerateIsRunningCmd() []string {
	args := []string{
		"-c",
		"ls -l",
	}

	return args
}

// Kill used to kill the mock.
func (l *MockArgs) GenerateKillCmd() []string {
	args := []string{
		"-c",
		"ls -l",
	}
	return args
}
