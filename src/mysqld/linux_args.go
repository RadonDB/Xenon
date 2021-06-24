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
	"path/filepath"
)

var (
	_ ArgsHandler = &LinuxArgs{}
)

const (
	bash       = "bash"
	mysqldsafe = "bin/mysqld_safe"
	mysqladmin = "bin/mysqladmin"
)

// LinuxArgs tuple.
type LinuxArgs struct {
	conf *config.BackupConfig
	ArgsHandler
}

// NewLinuxArgs creates new LinuxArgs.
func NewLinuxArgs(conf *config.BackupConfig) *LinuxArgs {
	return &LinuxArgs{
		conf: conf,
	}
}

// Start used to start mysqld.
func (l *LinuxArgs) GenerateStartCmd() []string {
	safe57 := filepath.Join(l.conf.Basedir, mysqldsafe)
	args := []string{
		"-c",
		fmt.Sprintf("%s --defaults-file=%s > /dev/null&", safe57, l.conf.DefaultsFile),
	}
	return args
}

// Stop used to stop the mysqld.
func (l *LinuxArgs) GenerateStopCmd() []string {
	admin57 := filepath.Join(l.conf.Basedir, mysqladmin)
	args := []string{
		"-c",
	}
	if l.conf.Passwd == "" {
		args = append(args, fmt.Sprintf("%s -h%s -u%s -P%d shutdown", admin57, l.conf.Host, l.conf.Admin, l.conf.Port))
	} else {
		args = append(args, fmt.Sprintf("%s -h%s -u%s -p%s -P%d shutdown", admin57, l.conf.Host, l.conf.Admin, l.conf.Passwd, l.conf.Port))
	}
	return args
}

// IsRunning used to check the mysqld is running or not.
func (l *LinuxArgs) GenerateIsRunningCmd() []string {
	// [m] is a trick to stop you picking up the actual grep process itself
	safe57 := fmt.Sprintf("[m]ysqld_safe --defaults-file=%s", l.conf.DefaultsFile)
	args := []string{
		"-c",
		fmt.Sprintf("ps aux | grep '%s' | wc -l", safe57),
	}
	return args
}

// Kill used to kill -9 the mysqld process.
func (l *LinuxArgs) GenerateKillCmd() []string {
	args := []string{
		"-c",
	}
	args = append(args,
		fmt.Sprintf("kill -9 $(ps aux | grep '[-]-defaults-file=%s' | awk '{print $2}')", l.conf.DefaultsFile))
	return args
}
