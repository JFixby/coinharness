// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package coinharness

import (
	"fmt"
	"github.com/jfixby/pin"
	"math"
	"time"
)

type RPCClientFactory interface {
	NewRPCConnection(config RPCConnectionConfig, handlers RPCClientNotificationHandlers) (RPCClient, error)
}

type RPCClient interface {
	NotifyBlocks() error
	Disconnect()
	Shutdown()
	GetPeerInfo() ([]PeerInfo, error)
	GetBlockCount() (int64, error)
	GetRawMempool() ([]Hash, error)
	AddNode(arguments *AddNodeArguments) error
	Internal() interface{}
	Generate(blocks uint32) ([]Hash, error)
	SendRawTransaction(tx CreatedTransactionTx, b bool) (Hash, error)
}
type PeerInfo struct {
	Addr string
}

type RPCClientNotificationHandlers interface {
}

// RPCConnection is a helper class wrapping rpcclient.Client for test calls
type RPCConnection struct {
	rpcClient        RPCClient
	MaxConnRetries   int
	isConnected      bool
	RPCClientFactory RPCClientFactory
}

// NewRPCConnection produces new instance of the RPCConnection
func NewRPCConnection(fact RPCClientFactory, config RPCConnectionConfig, maxConnRetries int, ntfnHandlers RPCClientNotificationHandlers) RPCClient {
	var client RPCClient
	var err error

	for i := 0; i < maxConnRetries; i++ {
		client, err = fact.NewRPCConnection(config, ntfnHandlers)
		if err != nil {
			fmt.Println("err: " + err.Error())
			time.Sleep(time.Duration(math.Log(float64(i+3))) * 50 * time.Millisecond)
			continue
		}
		break
	}
	if client == nil {
		pin.ReportTestSetupMalfunction(fmt.Errorf("client connection timedout"))
	}
	return client
}

// Connect switches RPCConnection into connected state establishing RPCConnection to the rpcConf target
func (client *RPCConnection) Connect(rpcConf RPCConnectionConfig, ntfnHandlers RPCClientNotificationHandlers) {
	if client.isConnected {
		pin.ReportTestSetupMalfunction(fmt.Errorf("%v is already connected", client.rpcClient))
	}
	client.isConnected = true
	rpcClient := NewRPCConnection(client.RPCClientFactory, rpcConf, client.MaxConnRetries, ntfnHandlers)
	err := rpcClient.NotifyBlocks()
	pin.CheckTestSetupMalfunction(err)
	client.rpcClient = rpcClient
}

// Disconnect switches RPCConnection into offline state
func (client *RPCConnection) Disconnect() {
	if !client.isConnected {
		pin.ReportTestSetupMalfunction(fmt.Errorf("%v is already disconnected", client))
	}
	client.isConnected = false
	client.rpcClient.Disconnect()
	client.rpcClient.Shutdown()
}

// IsConnected flags RPCConnection state
func (client *RPCConnection) IsConnected() bool {
	return client.isConnected
}

// Connection returns rpcclient.Client for API calls
func (client *RPCConnection) Connection() RPCClient {
	return client.rpcClient
}
