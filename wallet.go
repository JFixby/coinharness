// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package coinharness

// Wallet wraps optional test wallet implementations for different test setups
type Wallet interface {
	// Network returns current network of the wallet
	Network() Network

	// NewAddress returns a fresh address spendable by the wallet.
	NewAddress(args *NewAddressArgs) (Address, error)

	// Start wallet process
	Start(args *TestWalletStartArgs) error

	// Stops wallet process gently
	Stop()

	// Dispose releases all resources allocated by the wallet
	// This action is final (irreversible)
	Dispose() error

	// Sync block until the wallet has fully synced up to the desiredHeight
	Sync(desiredHeight int64) int64

	SyncedHeight() int64

	// ConfirmedBalance returns wallet balance
	ConfirmedBalance() CoinsAmount

	// CreateTransaction returns a fully signed transaction paying to the specified
	// outputs while observing the desired fee rate. The passed fee rate should be
	// expressed in satoshis-per-byte. The transaction being created can optionally
	// include a change output indicated by the Change boolean.
	CreateTransaction(args *CreateTransactionArgs) (CreatedTransactionTx, error)

	// SendOutputs creates, then sends a transaction paying to the specified output
	// while observing the passed fee rate. The passed fee rate should be expressed
	// in satoshis-per-byte.
	SendOutputs(args *SendOutputsArgs) (Hash, error)

	// UnlockOutputs unlocks any outputs which were previously locked due to
	// being selected to fund a transaction via the CreateTransaction method.
	UnlockOutputs(inputs []InputTx)

	// RPCClient returns node RPCConnection
	RPCClient() *RPCConnection

	GetNewAddress(accountName string) (Address, error)
	CreateNewAccount(accountName string) error
	ValidateAddress(address Address) (*ValidateAddressResult, error)
	//GetBalance(accountName string) (CoinsAmount, error)
}

type ValidateAddressResult struct {
	IsValid  bool
	IsMine   bool
	IsScript bool
	Account  string
}

// TestWalletFactory produces a new Wallet instance
type TestWalletFactory interface {
	// NewWallet is used by harness builder to setup a wallet component
	NewWallet(cfg *TestWalletConfig) Wallet
}

// TestWalletConfig bundles settings required to create a new wallet instance
type TestWalletConfig struct {
	Seed          Seed //[]byte // chainhash.HashSize + 4
	NodeRPCHost   string
	NodeRPCPort   int
	WalletRPCHost string
	WalletRPCPort int
	ActiveNet     ActiveNet
	WorkingDir    string

	NodeUser       string
	NodePassword   string
	WalletUser     string
	WalletPassword string
}

type SendOutputsArgs struct {
	Outputs []OutputTx
	FeeRate CoinsAmount
}

// CreateTransactionArgs bundles CreateTransaction() arguments to minimize diff
// in case a new argument for the function is added
type CreateTransactionArgs struct {
	Outputs   []OutputTx
	FeeRate   CoinsAmount
	Change    bool
	TxVersion int32
}

// NewAddressArgs bundles NewAddress() arguments to minimize diff
// in case a new argument for the function is added
type NewAddressArgs struct {
	Account string
}

// TestWalletStartArgs bundles Start() arguments to minimize diff
// in case a new argument for the function is added
type TestWalletStartArgs struct {
	NodeRPCCertFile          string
	ExtraArguments           map[string]interface{}
	DebugOutput              bool
	MaxSecondsToWaitOnLaunch int
	NodeRPCConfig            RPCConnectionConfig
}
