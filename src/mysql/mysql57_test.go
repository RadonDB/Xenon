/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package mysql

import (
	"testing"

	"config"
	"xbase/xlog"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestMysql57Handler(t *testing.T) {
	log := xlog.NewStdLog(xlog.Level(xlog.PANIC))
	conf := config.DefaultMysqlConfig()

	mysql := NewMysql(conf, QueryTimeout, log)
	want := new(Mysql57)
	want.SetQueryTimeout(QueryTimeout)
	got := mysql.mysqlHandler
	assert.Equal(t, want, got)
}
