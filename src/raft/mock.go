/*
 * Xenon
 *
 * Copyright 2018 The Xenon Authors.
 * Code is licensed under the GPLv3.
 *
 */

package raft

import (
	"config"
	"fmt"
	"model"
	"mysql"
	"os"
	"testing"
	"time"
	"xbase/common"
	"xbase/xlog"
	"xbase/xrpc"

	"github.com/stretchr/testify/assert"
)

var (
	shortHeartbeatTimeoutForTest = 100
	longHeartbeatTimeoutForTest  = 500
)

const (
	QueryTimeout    int    = 10000
	SemiSyncTimeout uint64 = 10000
)

func setupRPC(rpc *xrpc.Service, raft *Raft) {
	if err := rpc.RegisterService(raft.GetHARPC()); err != nil {
		raft.PANIC("server.rpc.RegisterService.HARPC.error[%+v]", err)
	}

	if err := rpc.RegisterService(raft.GetRaftRPC()); err != nil {
		raft.PANIC("server.rpc.RegisterService.RaftRPC.error[%+v]", err)
	}
}

// MockRaftsWithConfig mock.
func MockRaftsWithConfig(log *xlog.Log, conf *config.RaftConfig, port int, count int, idleStart int) ([]string, []*Raft, func()) {
	return mockRafts(log, conf, port, count, idleStart, false)
}

// MockRafts mock.
// If no idle nodes, set idleStart to -1
func MockRafts(log *xlog.Log, port int, count int, idleStart int) ([]string, []*Raft, func()) {
	conf := config.DefaultRaftConfig()
	conf.PurgeBinlogInterval = 1
	conf.CandidateWaitFor2Nodes = 1000
	conf.MetaDatadir = "/tmp/"

	return mockRafts(log, conf, port, count, idleStart, false)
}

// MockRaftsWithLong mock.
func MockRaftsWithLong(log *xlog.Log, port int, count int, idleStart int) ([]string, []*Raft, func()) {
	conf := config.DefaultRaftConfig()
	conf.PurgeBinlogInterval = 1
	conf.MetaDatadir = "/tmp/"

	return mockRafts(log, conf, port, count, idleStart, true)
}

func mockRafts(log *xlog.Log, conf *config.RaftConfig, port int, count int, idleStart int, islong bool) ([]string, []*Raft, func()) {
	ids := []string{}
	var raft *Raft
	rafts := []*Raft{}
	rpcs := []*xrpc.Service{}
	ip, _ := common.GetLocalIP()

	os.Remove("/tmp/peers.json")
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("%s:%d", ip, port+i)
		ids = append(ids, id)

		if islong {
			conf.HeartbeatTimeout = longHeartbeatTimeoutForTest
			conf.ElectionTimeout = longHeartbeatTimeoutForTest * 3
		} else {
			conf.HeartbeatTimeout = shortHeartbeatTimeoutForTest
			conf.ElectionTimeout = shortHeartbeatTimeoutForTest * 3
		}

		// setup mysql
		mysql57 := mysql.NewMysql(config.DefaultMysqlConfig(), QueryTimeout, log)
		mysql57.SetMysqlHandler(mysql.NewMockGTIDA())
		mysql57.PingStart()

		for i, id := range ids {
			if idleStart != -1 && i >= idleStart {
				raft = NewRaft(id, conf, SemiSyncTimeout, log, mysql57, IDLE)
			} else {
				raft = NewRaft(id, conf, SemiSyncTimeout, log, mysql57, FOLLOWER)
			}
		}

		// setup raft
		rafts = append(rafts, raft)

		// setup rpc
		rpc, err := xrpc.NewService(xrpc.Log(log),
			xrpc.ConnectionStr(id))
		if err != nil {
			log.Panic("raftRPC.NewService.error[%+v]", err)
		}
		setupRPC(rpc, raft)
		rpcs = append(rpcs, rpc)
		rpc.Start()
	}

	for _, raft := range rafts {
		for i, id := range ids {
			if idleStart != -1 && i >= idleStart {
				raft.AddIdlePeer(id)
			} else {
				raft.AddPeer(id)
			}
		}
	}

	return ids, rafts, func() {
		os.Remove("peers.json")
		for i, r := range rafts {
			rpcs[i].Stop()
			r.Stop()
		}
	}
}

// MockSetMysqlHandler used to set mysql repl hander for test.
func MockSetMysqlHandler(raft *Raft, h mysql.MysqlHandler) {
	raft.mysql.SetMysqlHandler(h)
}

// MockWaitMySQLPingTimeout used to wait mysql ping timeout.
func MockWaitMySQLPingTimeout() {
	pingTimeout := config.DefaultMysqlConfig().PingTimeout * 6
	time.Sleep(time.Millisecond * time.Duration(pingTimeout))
}

// MockWaitHeartBeatTimeout used to wait mysql ping timeout.
func MockWaitHeartBeatTimeout() {
	hbTimeout := config.DefaultRaftConfig().HeartbeatTimeout * 6
	time.Sleep(time.Millisecond * time.Duration(hbTimeout))
}

// MockWaitLeaderEggs mock.
// wait the leader eggs when leadernums >0
// if leadernums == 0, we just want to sleep for a heartbeat broadcast
func MockWaitLeaderEggs(rafts []*Raft, leadernums int) int {
	// wait
	if leadernums == 0 {
		// wait hearbeat broadcast
		time.Sleep(time.Millisecond * time.Duration(rafts[0].getElectionTimeout()*2))
		return -1
	}

	done := make(chan int, 1)
	maxRunTime := time.Duration(60) * time.Second
	go func() {
		for {
			nums := 0
			for i, raft := range rafts {
				if raft.getState() == LEADER {
					nums++
					if nums == leadernums {
						// wait hearbeat broadcast
						time.Sleep(time.Millisecond * time.Duration(rafts[0].getElectionTimeout()))
						done <- i
						return
					}
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	select {
	case <-time.After(maxRunTime):
		return -1
	case idx := <-done:
		return idx
	}
}

// MockStateTransition use to transfer the raft.state to state.
func MockStateTransition(raft *Raft, state State) {
	raft.setState(state)
	raft.loopFired()
}

// MockGetClient mock.
func MockGetClient(t *testing.T, svrConn string) (*xrpc.Client, func()) {
	client, err := xrpc.NewClient(svrConn, 100)
	assert.Nil(t, err)

	return client, func() {
		client.Close()
	}
}

// mock leader process heartbeat request who comes from the new leader
// nop here, so our leader state won't be changed(degrade)
func (r *Raft) mockLeaderProcessHeartbeatRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.ErrorInvalidRequest)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	r.INFO("mock.get.heartbeat.from[N:%v, V:%v, E:%v]", req.GetFrom(), req.GetViewID(), req.GetEpochID())

	return rsp
}

// mock leader process requestvote request from other candidate
// nop here, so our leader state won't be changed(degrade)
func (r *Raft) mockLeaderProcessRequestVoteRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.ErrorInvalidRequest)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()

	r.INFO("mock.get.requestvote.from[N:%v, V:%v, E:%v]", req.GetFrom(), req.GetViewID(), req.GetEpochID())

	return rsp
}

// mock leader send heartbeat request
// nop here, so other followers will start a new leader election
func (r *Raft) mockLeaderSendHeartbeat(mysqlDown *bool, c chan *model.RaftRPCResponse) {
	r.DEBUG("mock.send.nop.heartbeat.request")
}

// mock leader prpcess heartbeat response
func (r *Raft) mockLeaderProcessSendHeartbeatResponse(ackGranted *int, rsp *model.RaftRPCResponse) {
	r.DEBUG("mock.send.heartbeat.get.rsp[N:%v, V:%v, E:%v].retcode[%v]", rsp.GetFrom(), rsp.GetViewID(), rsp.GetEpochID(), rsp.RetCode)
}
