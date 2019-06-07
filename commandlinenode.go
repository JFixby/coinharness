// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cointest

import (
	"fmt"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
	"path/filepath"
)

type ConsoleCommandCook interface {
	CookArguments(ExtraArguments map[string]interface{}) map[string]interface{}
}

// CommandlineTestNode launches a new dcrd instance using command-line call.
// Implements harness.TestNode.
type CommandlineTestNode struct {
	// NodeExecutablePathProvider returns path to the dcrd executable
	NodeExecutablePathProvider commandline.ExecutablePathProvider

	p2pAddress string
	rpcConnect string
	profile    string
	debugLevel string
	appDir     string

	externalProcess *commandline.ExternalProcess

	rPCClient *RPCConnection

	miningAddress Address

	network                    Network
	RPCClientFactory           RPCClientFactory
	RPCConnectionConfigSpawner RPCConnectionConfigSpawner

	ConsoleCommandCook ConsoleCommandCook
}

// FullConsoleCommand returns the full console command used to
// launch external process of the node
func (node *CommandlineTestNode) FullConsoleCommand() string {
	return node.externalProcess.FullConsoleCommand()
}

// P2PAddress returns node p2p address
func (node *CommandlineTestNode) P2PAddress() string {
	return node.p2pAddress
}

// RPCClient returns node RPCConnection
func (node *CommandlineTestNode) RPCClient() *RPCConnection {
	return node.rPCClient
}

// CertFile returns file path of the .cert-file of the node
func (node *CommandlineTestNode) CertFile() string {
	return filepath.Join(node.appDir, "rpc.cert")
}

// KeyFile returns file path of the .key-file of the node
func (node *CommandlineTestNode) KeyFile() string {
	return filepath.Join(node.appDir, "rpc.key")
}

// Network returns current network of the node
func (node *CommandlineTestNode) Network() Network {
	return node.network
}

// IsRunning returns true if CommandlineTestNode is running external dcrd process
func (node *CommandlineTestNode) IsRunning() bool {
	return node.externalProcess.IsRunning()
}

// Start node process. Deploys working dir, launches dcrd using command-line,
// connects RPC client to the node.
func (node *CommandlineTestNode) Start(args *TestNodeStartArgs) {
	if node.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("CommandlineTestNode is already running"))
	}
	fmt.Println("Start node process...")
	pin.MakeDirs(node.appDir)

	node.miningAddress = args.MiningAddress

	exec := node.NodeExecutablePathProvider.Executable()
	node.externalProcess.CommandName = exec
	node.externalProcess.Arguments = commandline.ArgumentsToStringArray(
		node.ConsoleCommandCook.CookArguments(args.ExtraArguments),
	)
	node.externalProcess.Launch(args.DebugOutput)
	// Node RPC instance will create a cert file when it is ready for incoming calls
	pin.WaitForFile(node.CertFile(), 7)

	fmt.Println("Connect to node RPC...")
	node.rPCClient.Connect(node.RPCConnectionConfigSpawner.Spawn(), nil)
	fmt.Println("node RPC client connected.")
}

// Stop interrupts the running node process.
// Disconnects RPC client from the node, removes cert-files produced by the dcrd,
// stops dcrd process.
func (node *CommandlineTestNode) Stop() {
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
func (node *CommandlineTestNode) Dispose() error {
	if node.IsRunning() {
		node.Stop()
	}
	return nil
}
