// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cointest

import (
	"fmt"
	"os"
)

// Harness provides a unified platform for creating RPC-driven
// integration tests involving node and wallet executables.
// The active TestNodeServer and TestWallet will typically be
// run in regression net mode to allow for easy generation of test blockchains.
type Harness struct {
	Name string

	Node   TestNode
	Wallet TestWallet

	WorkingDir string

	MiningAddress Address
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
