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
	GetBalance(accountName string) (*GetBalanceResult, error)

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
	UnlockOutputs(inputs []InputTx) error

	// RPCClient returns node RPCConnection
	RPCClient() *RPCConnection

	GetNewAddress(accountName string) (Address, error)

	CreateNewAccount(accountName string) error

	ValidateAddress(address Address) (*ValidateAddressResult, error)

	WalletUnlock(password string, timeout int64) error

	WalletLock() error

	WalletInfo() (*WalletInfoResult, error)
}

type WalletInfoResult struct {
	Unlocked         bool
	DaemonConnected  bool
	TxFee            float64
	TicketFee        float64
	TicketPurchasing bool
	VoteBits         uint16
	VoteBitsExtended string
	VoteVersion      uint32
	Voting           bool
}

type GetBalanceResult struct {
	Balances                     []GetAccountBalanceResult
	BlockHash                    Hash
	TotalImmatureCoinbaseRewards CoinsAmount
	TotalImmatureStakeGeneration CoinsAmount
	TotalLockedByTickets         CoinsAmount
	TotalSpendable               CoinsAmount
	CumulativeTotal              CoinsAmount
	TotalUnconfirmed             CoinsAmount
	TotalVotingAuthority         CoinsAmount
}

// GetAccountBalanceResult models the account data from the getbalance command.
type GetAccountBalanceResult struct {
	AccountName             string
	ImmatureCoinbaseRewards CoinsAmount
	ImmatureStakeGeneration CoinsAmount
	LockedByTickets         CoinsAmount
	Spendable               CoinsAmount
	Total                   CoinsAmount
	Unconfirmed             CoinsAmount
	VotingAuthority         CoinsAmount
}

type ValidateAddressResult struct {
	IsValid      bool
	Address      string
	IsMine       bool
	IsWatchOnly  bool
	IsScript     bool
	PubKeyAddr   string
	PubKey       string
	IsCompressed bool
	Account      string
	Addresses    []string
	Hex          string
	Script       string
	SigsRequired int32
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
