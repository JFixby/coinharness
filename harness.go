package coinharness

import (
	"fmt"
	"os"
)

// Harness provides a unified platform for creating RPC-driven
// integration tests involving node and wallet executables.
// The active TestNodeServer and Wallet will typically be
// run in regression Net mode to allow for easy generation of test blockchains.
type Harness struct {
	Name string

	Node   Node
	Wallet Wallet

	WorkingDir string

	MiningAddress Address
}

// WalletRPCClient manages access to the RPCClient,
// test cases suppose to use it when the need access to the Wallet RPC
func (harness *Harness) WalletRPCClient() RPCClient {
	return harness.Wallet.RPCClient().rpcClient
}

// NodeRPCClient manages access to the RPCClient,
// test cases suppose to use it when the need access to the node RPC
func (harness *Harness) NodeRPCClient() RPCClient {
	return harness.Node.RPCClient().rpcClient
}

// DeleteWorkingDir removes harness working directory
func (harness *Harness) DeleteWorkingDir() error {
	dir := harness.WorkingDir
	fmt.Println("delete: " + dir)
	err := os.RemoveAll(dir)
	return err
}

// P2PAddress is a shortcut to the harness.Node.P2PAddress()
func (harness *Harness) P2PAddress() string {
	return harness.Node.P2PAddress()
}
