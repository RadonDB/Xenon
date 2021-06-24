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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinuxStartArgs(t *testing.T) {
	conf := config.DefaultBackupConfig()
	linuxargs := NewLinuxArgs(conf)
	want := `-c /u01/mysql_20160606/bin/mysqld_safe --defaults-file=/etc/my3306.cnf > /dev/null&`
	got := strings.Join(linuxargs.GenerateStartCmd(), " ")
	assert.Equal(t, want, got)
}

func TestLinuxStopArgs(t *testing.T) {
	conf := config.DefaultBackupConfig()
	linuxargs := NewLinuxArgs(conf)

	// 1. passwords is null
	{
		want := `-c /u01/mysql_20160606/bin/mysqladmin -hlocalhost -uroot -P3306 shutdown`
		got := strings.Join(linuxargs.GenerateStopCmd(), " ")
		assert.Equal(t, want, got)
	}

	// 2. with passwords
	{
		conf.Passwd = `ddd"`
		want := `-c /u01/mysql_20160606/bin/mysqladmin -hlocalhost -uroot -pddd" -P3306 shutdown`
		got := strings.Join(linuxargs.GenerateStopCmd(), " ")
		assert.Equal(t, want, got)
	}
}

func TestLinuxIsRunningArgs(t *testing.T) {
	linuxargs := NewLinuxArgs(config.DefaultBackupConfig())
	want := `-c ps aux | grep '[m]ysqld_safe --defaults-file=/etc/my3306.cnf' | wc -l`
	got := strings.Join(linuxargs.GenerateIsRunningCmd(), " ")
	assert.Equal(t, want, got)
}

func TestLinuxKillArgs(t *testing.T) {
	linuxargs := NewLinuxArgs(config.DefaultBackupConfig())
	want := `-c kill -9 $(ps aux | grep '[-]-defaults-file=/etc/my3306.cnf' | awk '{print $2}')`
	got := strings.Join(linuxargs.GenerateKillCmd(), " ")
	assert.Equal(t, want, got)
}
