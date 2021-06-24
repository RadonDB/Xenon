/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package server

import (
	"config"
	"mysql"
	"mysqld"
	"os"
	"os/signal"
	"raft"
	"runtime"
	"syscall"
	"time"
	"xbase/xlog"
	"xbase/xrpc"
)

// execute timeout: 60s
const TIMEOUT int = 60 * 1000

type RPCS struct {
	NodeRPC   *NodeRPC
	ServerRPC *ServerRPC
	UserRPC   *UserRPC
	HARPC     *raft.HARPC
	RaftRPC   *raft.RaftRPC
	MysqldRPC *mysqld.MysqldRPC
	BackupRPC *mysqld.BackupRPC
	MysqlRPC  *mysql.MysqlRPC
}

type Server struct {
	log    *xlog.Log
	mysqld *mysqld.Mysqld
	mysql  *mysql.Mysql
	raft   *raft.Raft
	conf   *config.Config
	rpc    *xrpc.Service
	rpcs   RPCS
	begin  time.Time
}

func NewServer(conf *config.Config, log *xlog.Log, initState raft.State) *Server {
	s := &Server{
		log:  log,
		conf: conf,
	}

	s.mysqld = mysqld.NewMysqld(conf.Backup, log)
	s.mysql = mysql.NewMysql(conf.Mysql, conf.Raft.ElectionTimeout, log)
	s.raft = raft.NewRaft(conf.Server.Endpoint, conf.Raft, conf.Mysql.SemiSyncTimeoutForTwoNodes, log, s.mysql, initState)
	rpc, err := xrpc.NewService(xrpc.Log(log),
		xrpc.ConnectionStr(conf.Server.Endpoint))
	if err != nil {
		log.Panic("server.rpc.NewService.error[%v]", err)
	}
	s.rpc = rpc
	return s
}

func (s *Server) Init() {
	s.setupMysqld()
	s.setupMysql()
	s.setupRPC()
}

// setupMysqld used to start mysqld and wait for it works
func (s *Server) setupMysqld() {
	log := s.log
	log.Info("server.prepare.setup.mysqlserver")
	if err := s.mysqld.StartMysqld(); err != nil {
		log.Error("server.mysqlserver.start.error[%v]", err)
		return
	}
	log.Info("server.mysqlserver.setup.done")
}

// setupMysql used to create replication user where not exists
func (s *Server) setupMysql() {
	log := s.log
	log.Info("server.mysql.wait.for.work[maxwait:60s]")
	if err := s.mysql.WaitMysqlWorks(TIMEOUT); err != nil {
		log.Error("server.mysql.WaitMysqlWorks.error[%v]", err)
		return
	}

	gtid, _ := s.mysql.GetGTID()
	log.Info("server.mysql.gtid:%+v", gtid)

	log.Info("server.mysql.set.to.READONLY")
	if err := s.mysql.SetReadOnly(); err != nil {
		log.Error("server.mysql.SetReadOnly.error[%+v]", err)
		return
	}

	log.Info("server.mysql.start.slave")
	if err := s.mysql.StartSlave(); err != nil {
		log.Error("server.mysql.start.slave.error[%+v]", err)
	}

	log.Info("server.mysql.check.replication.user...")
	ret, err := s.mysql.CheckUserExists(s.conf.Mysql.ReplUser, "%")
	if err != nil {
		log.Error("server.mysql.CheckUserExists.error[%+v]", err)
		return
	}
	if !ret {
		log.Info("setupMysql.server.mysql.prepare.to.create.replication.user[%v]", s.conf.Mysql.ReplUser)
		user := s.conf.Mysql.ReplUser
		pwd := s.conf.Mysql.ReplPasswd
		if err = s.mysql.CreateReplUserWithoutBinlog(user, pwd); err != nil {
			log.Error("server.mysql.create.replication.user[%v, %v].error[%+v]", user, pwd, err)
		}
	}
	log.Info("server.mysql.setup.done")
}

// setupRPC used to setup rpc handlers
func (s *Server) setupRPC() {
	log := s.log
	log.Info("server.prepare.setup.RPC")
	s.rpcs.NodeRPC = s.GetNodeRPC()
	s.rpcs.ServerRPC = s.GetServerRPC()
	s.rpcs.UserRPC = s.GetUserRPC()
	s.rpcs.RaftRPC = s.raft.GetRaftRPC()
	s.rpcs.HARPC = s.raft.GetHARPC()
	s.rpcs.MysqldRPC = s.mysqld.GetMysqldRPC()
	s.rpcs.BackupRPC = s.mysqld.GetBackupRPC()
	s.rpcs.MysqlRPC = s.mysql.GetMysqlRPC()

	if err := s.rpc.RegisterService(s.rpcs.NodeRPC); err != nil {
		log.Panic("server.rpc.RegisterService.NodeRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.ServerRPC); err != nil {
		log.Panic("server.rpc.RegisterService.ServerRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.UserRPC); err != nil {
		log.Panic("server.rpc.RegisterService.UserRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.HARPC); err != nil {
		log.Panic("server.rpc.RegisterService.HARPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.RaftRPC); err != nil {
		log.Panic("server.rpc.RegisterService.RaftRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.MysqldRPC); err != nil {
		log.Panic("server.rpc.RegisterService.MysqldRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.BackupRPC); err != nil {
		log.Panic("server.rpc.RegisterService.BackupRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.MysqlRPC); err != nil {
		log.Panic("server.rpc.RegisterService.MysqlRPC.error[%+v]", err)
	}
	log.Info("server.RPC.setup.done")
}

// start rpc-raft and event&&state-loop
func (s *Server) Start() {
	log := s.log
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, false)]
			log.Error("server.got.panic[%v:%s]", r, buf)
		}
	}()

	s.mysqld.MonitorStart()
	s.mysql.PingStart()
	if err := s.raft.Start(); err != nil {
		log.Panic("server.raft.start.error[%+v]", err)
	}
	if err := s.rpc.Start(); err != nil {
		log.Panic("server.rpc.start.error[%+v]", err)
	}
	s.updateUptime()
	log.Info("server.start.success...")
}

func (s *Server) updateUptime() {
	s.begin = time.Now()
}

func (s *Server) Shutdown() {
	s.log.Info("server.prepare.to.shutdown")
	s.rpc.Stop()
	s.raft.Stop()
	s.mysql.PingStop()
	s.mysqld.MonitorStop()
	s.log.Info("server.shutdown.done")
}

// waits for os signal
func (s *Server) Wait() {
	ossig := make(chan os.Signal, 1)
	signal.Notify(ossig,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP)
	s.log.Info("server.signal:%+v", <-ossig)
	s.Shutdown()
}

func (s *Server) GetState() raft.State {
	return s.raft.GetState()
}

func (s *Server) Address() string {
	return s.conf.Server.Endpoint
}
