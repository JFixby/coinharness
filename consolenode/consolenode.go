// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package consolenode

import (
	"fmt"
	"github.com/jfixby/cointest"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
	"net"
	"path/filepath"
	"strconv"
)

type NewConsoleNodeArgs struct {
	ClientFac                  coinharness.RPCClientFactory
	ConsoleCommandCook         ConsoleCommandCook
	NodeExecutablePathProvider commandline.ExecutablePathProvider
	RpcUser                    string
	RpcPass                    string
	AppDir                     string
	ActiveNet                  coinharness.Network
	P2PHost                    string
	NodeRPCHost                string
	P2PPort                    int
	NodeRPCPort                int
}

func NewConsoleNode(args *NewConsoleNodeArgs) *ConsoleNode {
	pin.AssertNotNil("NodeExecutablePathProvider", args.NodeExecutablePathProvider)
	pin.AssertNotNil("ActiveNet", args.ActiveNet)
	pin.AssertNotNil("ClientFac", args.ClientFac)

	node := &ConsoleNode{
		p2pAddress:                 net.JoinHostPort(args.P2PHost, strconv.Itoa(args.P2PPort)),
		rpcListen:                  net.JoinHostPort(args.NodeRPCHost, strconv.Itoa(args.NodeRPCPort)),
		rpcUser:                    args.RpcUser,
		rpcPass:                    args.RpcPass,
		appDir:                     args.AppDir,
		endpoint:                   "ws",
		rPCClient:                  &coinharness.RPCConnection{MaxConnRetries: 20, RPCClientFactory: args.ClientFac},
		NodeExecutablePathProvider: args.NodeExecutablePathProvider,
		network:                    args.ActiveNet,
		ConsoleCommandCook:         args.ConsoleCommandCook,
	}
	return node
}

// ConsoleNode launches a new dcrd instance using command-line call.
// Implements harness.TestNode.
type ConsoleNode struct {
	// NodeExecutablePathProvider returns path to the dcrd executable
	NodeExecutablePathProvider commandline.ExecutablePathProvider

	rpcUser    string
	rpcPass    string
	p2pAddress string
	rpcListen  string
	rpcConnect string
	profile    string
	debugLevel string
	appDir     string
	endpoint   string

	externalProcess commandline.ExternalProcess

	rPCClient *coinharness.RPCConnection

	miningAddress coinharness.Address

	network coinharness.Network

	ConsoleCommandCook ConsoleCommandCook

	debugOutput bool
	extraArgs   map[string]interface{}
}

type ConsoleCommandParams struct {
	ExtraArguments map[string]interface{}
	RpcUser        string
	RpcPass        string
	RpcConnect     string
	RpcListen      string
	P2pAddress     string
	AppDir         string
	DebugLevel     string
	Profile        string
	CertFile       string
	KeyFile        string
	MiningAddress  coinharness.Address
	Network        coinharness.Network
}

type ConsoleCommandCook interface {
	CookArguments(par *ConsoleCommandParams) map[string]interface{}
}

// RPCConnectionConfig produces a new connection config instance for RPC client
func (node *ConsoleNode) RPCConnectionConfig() coinharness.RPCConnectionConfig {
	return coinharness.RPCConnectionConfig{
		Host:            node.rpcListen,
		Endpoint:        node.endpoint,
		User:            node.rpcUser,
		Pass:            node.rpcPass,
		CertificateFile: node.CertFile(),
	}
}

// FullConsoleCommand returns the full console command used to
// launch external process of the node
func (node *ConsoleNode) FullConsoleCommand() string {
	return node.externalProcess.FullConsoleCommand()
}

func (node *ConsoleNode) SetExtraArguments(NodeExtraArguments map[string]interface{}) {
	if node.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("Unable to set parameter, ConsoleNode is already running"))
	}

	node.extraArgs = NodeExtraArguments
}

func (node *ConsoleNode) SetDebugNodeOutput(d bool) {
	if node.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("Unable to set parameter, ConsoleNode is already running"))
	}
	node.debugOutput = d
}

func (node *ConsoleNode) SetMiningAddress(address coinharness.Address) {
	if node.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("Unable to set parameter, ConsoleNode is already running"))
	}

	node.miningAddress = address
}

// P2PAddress returns node p2p address
func (node *ConsoleNode) P2PAddress() string {
	return node.p2pAddress
}

// RPCClient returns node RPCConnection
func (node *ConsoleNode) RPCClient() *coinharness.RPCConnection {
	return node.rPCClient
}

// CertFile returns file path of the .cert-file of the node
func (node *ConsoleNode) CertFile() string {
	return filepath.Join(node.appDir, "rpc.cert")
}

// KeyFile returns file path of the .key-file of the node
func (node *ConsoleNode) KeyFile() string {
	return filepath.Join(node.appDir, "rpc.key")
}

// Network returns current network of the node
func (node *ConsoleNode) Network() coinharness.Network {
	return node.network
}

// IsRunning returns true if ConsoleNode is running external dcrd process
func (node *ConsoleNode) IsRunning() bool {
	return node.externalProcess.IsRunning()
}

// Start node process. Deploys working dir, launches dcrd using command-line,
// connects RPC client to the node.
func (node *ConsoleNode) Start() {
	if node.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("ConsoleNode is already running"))
	}
	fmt.Println("Start node process...")
	pin.MakeDirs(node.appDir)

	exec := node.NodeExecutablePathProvider.Executable()
	node.externalProcess.CommandName = exec

	consoleCommandParams := &ConsoleCommandParams{
		ExtraArguments: node.extraArgs,
		RpcUser:        node.rpcUser,
		RpcPass:        node.rpcPass,
		RpcConnect:     node.rpcConnect,
		RpcListen:      node.rpcListen,
		P2pAddress:     node.P2PAddress(),
		AppDir:         node.appDir,
		DebugLevel:     node.debugLevel,
		Profile:        node.profile,
		CertFile:       node.CertFile(),
		KeyFile:        node.KeyFile(),
		MiningAddress:  node.miningAddress,
		Network:        node.network,
	}

	node.externalProcess.Arguments = commandline.ArgumentsToStringArray(
		node.ConsoleCommandCook.CookArguments(consoleCommandParams),
	)
	node.externalProcess.Launch(node.debugOutput)
	// Node RPC instance will create a cert file when it is ready for incoming calls
	pin.WaitForFile(node.CertFile(), 7)

	fmt.Println("Connect to node RPC...")
	cfg := node.RPCConnectionConfig()
	node.rPCClient.Connect(cfg, nil)
	fmt.Println("node RPC client connected.")
}

// Stop interrupts the running node process.
// Disconnects RPC client from the node, removes cert-files produced by the dcrd,
// stops dcrd process.
func (node *ConsoleNode) Stop() {
	if !node.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("node is not running"))
	}

	if node.rPCClient.IsConnected() {
		fmt.Println("Disconnect from node RPC...")
		node.rPCClient.Disconnect()
	}

	fmt.Println("Stop node process...")
	err := node.externalProcess.Stop()
	pin.CheckTestSetupMalfunction(err)

	// Delete files, RPC servers will recreate them on the next launch sequence
	pin.DeleteFile(node.CertFile())
	pin.DeleteFile(node.KeyFile())
}

// Dispose simply stops the node process if running
func (node *ConsoleNode) Dispose() error {
	if node.IsRunning() {
		node.Stop()
	}
	return nil
}
