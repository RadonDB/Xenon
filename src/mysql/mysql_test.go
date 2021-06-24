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
	"model"
	"testing"
	"time"
	"xbase/common"
	"xbase/xlog"

	"github.com/stretchr/testify/assert"
)

func TestMysql(t *testing.T) {
	// log
	log := xlog.NewStdLog(xlog.Level(xlog.PANIC))

	port := common.RandomPort(8000, 9000)
	_, mysql, cleanup := MockMysql(log, port, NewMockGTIDA())
	defer cleanup()

	time.Sleep(time.Duration(config.DefaultMysqlConfig().PingTimeout*2) * time.Millisecond)
	got := mysql.GetState()
	want := model.MysqlAlive
	assert.Equal(t, want, got)
	mysql.PingStop()
}

func TestStateDead(t *testing.T) {
	// log
	log := xlog.NewStdLog(xlog.Level(xlog.PANIC))
	port := common.RandomPort(8000, 9000)
	_, mysql, cleanup := MockMysql(log, port, NewMockGTIDPingError())
	defer cleanup()

	time.Sleep(time.Duration(config.DefaultMysqlConfig().PingTimeout*2) * time.Millisecond)
	got := mysql.GetState()
	want := model.MysqlDead
	assert.Equal(t, want, got)
	mysql.PingStop()
}

func TestCreateReplUser(t *testing.T) {
	// log
	log := xlog.NewStdLog(xlog.Level(xlog.PANIC))
	port := common.RandomPort(8000, 9000)
	_, mysql, cleanup := MockMysqlReplUser(log, port, NewMockGTIDA())
	defer cleanup()

	time.Sleep(time.Duration(config.DefaultMysqlConfig().PingTimeout*2) * time.Millisecond)
	got := mysql.GetState()
	want := model.MysqlAlive
	assert.Equal(t, want, got)
	mysql.PingStop()
}

/*
// TEST EFFECTS:
// test GTIDGreaterThan function
//
// TEST PROCESSES:
// 1. set mock function
// 2. greater than a
// 3. not greater than a
func TestMysqlGTIDGreatThan(t *testing.T) {
	// log
	log := xlog.NewStdLog(xlog.Level(xlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, QueryTimeout, log)

	// Set mock functions
	mysql.SetMysqlHandler(new(MockGTIDB))

	// start ping
	mysql.PingStart()

	// wait for ping
	time.Sleep(time.Duration(conf.PingTimeout*2) * time.Millisecond)

	// 1. greater than a
	a := model.GTID{Master_Log_File: "mysql-bin.000001",
		Read_Master_Log_Pos: 121}

	want := true
	got, _ := mysql.SlaveGTIDGreaterThan(&a)
	assert.Equal(t, want, got)

	// 2. greater than a
	a = model.GTID{Master_Log_File: "",
		Read_Master_Log_Pos: 0}
	want = true
	got, _ = mysql.SlaveGTIDGreaterThan(&a)
	assert.Equal(t, want, got)

	// 3. not greater than a
	a = model.GTID{Master_Log_File: "mysql-bin.000002",
		Read_Master_Log_Pos: 0}
	want = false
	got, _ = mysql.SlaveGTIDGreaterThan(&a)
	assert.Equal(t, want, got)

	// 4. nil compare: not greater than
	// set mock: this mock sets  null GTID
	mysql.SetMysqlHandler(new(MockGTIDA))

	// wait for ping
	time.Sleep(time.Duration(conf.PingTimeout*2) * time.Millisecond)

	a = model.GTID{Master_Log_File: "",
		Read_Master_Log_Pos: 0}
	want = false
	mysql.SetMysqlHandler(new(MockGTIDB))
	assert.Equal(t, want, got)
}
*/
