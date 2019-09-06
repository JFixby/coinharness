// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package coinharness

type StartNodeArgs struct {
	DebugOutput bool
}

// Node wraps optional test node implementations for different test setups
type Node interface {
	// Network returns current network of the node
	Network() Network

	// Start node process
	Start(args *StartNodeArgs)

	// Stop node process
	Stop()

	// Dispose releases all resources allocated by the node
	// This action is final (irreversible)
	Dispose() error

	// CertFile returns file path of the .cert-file of the node
	CertFile() string

	// RPCConnectionConfig produces a new connection config instance for RPC client
	RPCConnectionConfig() RPCConnectionConfig

	// RPCClient returns node RPCConnection
	RPCClient() *RPCConnection

	// P2PAddress returns node p2p address
	P2PAddress() string

	SetMiningAddress(address Address)
	SetExtraArguments(NodeExtraArguments map[string]interface{})
}

// TestNodeFactory produces a new Node instance
type TestNodeFactory interface {
	// NewNode is used by harness builder to setup a node component
	NewNode(cfg *TestNodeConfig) Node
}

// TestNodeConfig bundles settings required to create a new node instance
type TestNodeConfig struct {
	ActiveNet ActiveNet

	WorkingDir string

	P2PHost string
	P2PPort int

	NodeRPCHost string
	NodeRPCPort int
}
