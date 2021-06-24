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
	"model"
	"strconv"
	"strings"
	"sync"
	"time"
	"xbase/common"
	"xbase/xlog"
)

// Mysqld tuple.
type Mysqld struct {
	conf           *config.BackupConfig
	log            *xlog.Log
	cmd            common.Command
	backup         *Backup
	monitorTicker  *time.Ticker
	monitorRunning bool
	mutex          sync.RWMutex
	status         model.MYSQLD_STATUS
	stats          model.MysqldStats
	argsHandler    ArgsHandler
}

// NewMysqld creates the new Mysqld.
func NewMysqld(conf *config.BackupConfig, log *xlog.Log) *Mysqld {
	return &Mysqld{
		conf:        conf,
		log:         log,
		cmd:         common.NewLinuxCommand(log),
		backup:      NewBackup(conf, log),
		status:      model.MYSQLD_NOTRUNNING,
		argsHandler: NewLinuxArgs(conf),
	}
}

// SetArgsHandler used to set the args handler.
func (m *Mysqld) SetArgsHandler(h ArgsHandler) {
	m.argsHandler = h
}

// StartMysqld used to start mysql using mysqld_safe.
func (m *Mysqld) StartMysqld() error {
	log := m.log

	log.Warning("mysqld.prepare.to.start...")
	if m.isMysqldRunning() {
		log.Warning("mysqld.already.running...")
		return nil
	}

	timeout := 3000
	args := m.argsHandler.GenerateStartCmd()
	if _, err := m.cmd.RunCommandWithTimeout(timeout, bash, args); err != nil {
		log.Error("mysqld.start..error[%+v]", err)
		return err
	}
	log.Warning("mysqld.start.done...")
	m.IncMysqldStarts()
	return nil
}

// StopMysqld used to shutdown mysqld using mysqldadmin.
func (m *Mysqld) StopMysqld() error {
	log := m.log

	log.Warning("mysqld.prepare.to.shutdown...")
	if !m.isMysqldRunning() {
		return nil
	}

	timeout := 5000
	args := m.argsHandler.GenerateStopCmd()
	if _, err := m.cmd.RunCommandWithTimeout(timeout, bash, args); err != nil {
		log.Error("mysqld.stop.mysqld.error[%+v]", err)
		return err
	}
	m.setStatus(model.MYSQLD_SHUTDOWNING)
	m.IncMysqldStops()

	log.Warning("mysqld.shutdown.done...")
	return nil
}

// KillMysqld is used to shutdown mysql before we rebuild it.
func (m *Mysqld) KillMysqld() error {
	timeout := 3000
	log := m.log

	args := m.argsHandler.GenerateKillCmd()
	log.Warning("mysqld.prepare.to.kill[%v]...", args)
	if _, err := m.cmd.RunCommandWithTimeout(timeout, bash, args); err != nil {
		log.Error("mysqld.kill.mysqld.error[%+v]", err)
		return err
	}
	m.setStatus(model.MYSQLD_SHUTDOWNING)
	m.IncMysqldStops()
	log.Warning("mysqld.kill.done...")
	return nil
}

// check the mysqld_safe --defaults-file=[] process is running.
func (m *Mysqld) isMysqldRunning() bool {
	log := m.log

	args := m.argsHandler.GenerateIsRunningCmd()
	outs, err := m.cmd.RunCommand(bash, args)
	if err != nil {
		log.Error("mysqld57.isMysqldRunning.error[%v:%+v]", outs, err)
		return false
	}
	running, err := strconv.Atoi(strings.TrimSpace(outs))
	if err != nil {
		log.Error("isMysqldRunning.error[%+v]", err)
		return true
	}
	return (running > 0)
}

func (m *Mysqld) monitor() {
	if !m.isMysqldRunning() {
		m.setStatus(model.MYSQLD_NOTRUNNING)
		m.log.Error("mysqld_safe.is.dead.prepare.to.start.it...")
		m.StartMysqld()
	} else {
		m.setStatus(model.MYSQLD_ISRUNNING)
	}
}

// MonitorStart used to monite mysqld_safe is running or not.
func (m *Mysqld) MonitorStart() {
	if m.monitorRunning {
		return
	}

	// create ticker
	m.monitorTicker = common.NormalTicker(m.conf.MysqldMonitorInterval)
	go func() {
		for range m.monitorTicker.C {
			m.monitor()
		}
	}()
	m.monitorRunning = true
	m.IncMonitorStarts()
	m.log.Info("mysqld[%v].monitor.start...", m.conf.DefaultsFile)
}

// MonitorStop used to stop the monitor.
func (m *Mysqld) MonitorStop() {
	if !m.monitorRunning {
		return
	}

	m.monitorTicker.Stop()
	m.monitorRunning = false

	// set mysqld status to UNKNOW
	m.setStatus(model.MYSQLD_UNKNOW)
	m.IncMonitorStops()
	m.log.Info("mysqld[%v].monitor.stop...", m.conf.DefaultsFile)
}

func (m *Mysqld) setStatus(s model.MYSQLD_STATUS) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.status = s
}

func (m *Mysqld) getStatus() model.MYSQLD_STATUS {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.status
}

func (m *Mysqld) getMonitorInfo() string {
	if m.monitorRunning {
		return "ON"
	}
	return "OFF"
}

func (m *Mysqld) getMysqldInfo() string {
	if m.monitorRunning {
		return string(m.getStatus())
	}
	return "UNKNOW"
}
