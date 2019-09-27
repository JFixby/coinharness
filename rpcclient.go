package coinharness

import (
	"encoding/json"
	"fmt"
	"github.com/jfixby/coin"
	"github.com/jfixby/pin"
	"math"
	"time"
)

type RPCClientFactory interface {
	NewRPCConnection(config RPCConnectionConfig, handlers *NotificationHandlers) (RPCClient, error)
}

type RPCClient interface {
	NotifyBlocks() error
	Disconnect()
	Shutdown()
	GetPeerInfo() ([]PeerInfo, error)
	GetBlockCount() (int64, error)
	GetRawMempool(command interface{}) ([]Hash, error)
	AddNode(arguments *AddNodeArguments) error
	Internal() interface{}
	Generate(blocks uint32) ([]Hash, error)
	SendRawTransaction(tx *MessageTx, b bool) (Hash, error)
	GetNewAddress(accountName string) (Address, error)
	GetBuildVersion() (BuildVersion, error)
	GetBestBlock() (Hash, int64, error)
	ValidateAddress(address Address) (*ValidateAddressResult, error)
	CreateNewAccount(accountName string) error
	GetBalance() (*GetBalanceResult, error)
	ListUnspent() ([]*Unspent, error)

	WalletUnlock(walletPassphrase string, timeout int64) error
	WalletLock() error
	WalletInfo() (*WalletInfoResult, error)
	GetBlock(hash Hash) (*MsgBlock, error)
	SubmitBlock(block Block) error
	LoadTxFilter(b bool, addresses []Address) error
	ListAccounts() (map[string]coin.Amount, error)
}

// Unspent models a successful response from the listunspent request.
type Unspent struct {
	TxID          string
	Vout          uint32
	Tree          int8
	TxType        int
	Address       string
	Account       string
	ScriptPubKey  string
	RedeemScript  string
	Amount        coin.Amount
	Confirmations int64
	Spendable     bool
}

type MsgBlock struct {
	Transactions []*MessageTx
}

type Block interface {
}

type PeerInfo struct {
	Addr string
}

type BuildVersion interface {
	VersionString() string
}

// RPCConnection is a helper class wrapping rpcclient.Client for test calls
type RPCConnection struct {
	rpcClient        RPCClient
	MaxConnRetries   int
	isConnected      bool
	RPCClientFactory RPCClientFactory
}

// NewRPCConnection produces new instance of the RPCConnection
func NewRPCConnection(fact RPCClientFactory, config RPCConnectionConfig, maxConnRetries int, ntfnHandlers *NotificationHandlers) RPCClient {
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
func (client *RPCConnection) Connect(rpcConf RPCConnectionConfig, ntfnHandlers *NotificationHandlers) {
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

// NotificationHandlers defines callback function pointers to invoke with
// notifications.  Since all of the functions are nil by default, all
// notifications are effectively ignored until their handlers are set to a
// concrete callback.
//
// NOTE: Unless otherwise documented, these handlers must NOT directly call any
// blocking calls on the client instance since the input reader goroutine blocks
// until the callback has completed.  Doing so will result in a deadlock
// situation.
type NotificationHandlers struct {
	// OnClientConnected is invoked when the client connects or reconnects
	// to the RPC server.  This callback is run async with the rest of the
	// notification handlers, and is safe for blocking client requests.
	OnClientConnected func()

	// OnBlockConnected is invoked when a block is connected to the longest
	// (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.
	OnBlockConnected func(blockHeader []byte, transactions [][]byte)

	// OnBlockDisconnected is invoked when a block is disconnected from the
	// longest (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.
	OnBlockDisconnected func(blockHeader []byte)

	// OnRelevantTxAccepted is invoked when an unmined transaction passes
	// the client's transaction filter.
	OnRelevantTxAccepted func(transaction []byte)

	// OnReorganization is invoked when the blockchain begins reorganizing.
	// It will only be invoked if a preceding call to NotifyBlocks has been
	// made to register for the notification and the function is non-nil.
	OnReorganization func(oldHash Hash, oldHeight int32,
		newHash Hash, newHeight int32)

	// OnWinningTickets is invoked when a block is connected and eligible tickets
	// to be voted on for this chain are given.  It will only be invoked if a
	// preceding call to NotifyWinningTickets has been made to register for the
	// notification and the function is non-nil.
	OnWinningTickets func(blockHash Hash,
		blockHeight int64,
		tickets []Hash)

	// OnSpentAndMissedTickets is invoked when a block is connected to the
	// longest (best) chain and tickets are spent or missed.  It will only be
	// invoked if a preceding call to NotifySpentAndMissedTickets has been made to
	// register for the notification and the function is non-nil.
	OnSpentAndMissedTickets func(hash Hash,
		height int64,
		stakeDiff int64,
		tickets map[Hash]bool)

	// OnNewTickets is invoked when a block is connected to the longest (best)
	// chain and tickets have matured to become active.  It will only be invoked
	// if a preceding call to NotifyNewTickets has been made to register for the
	// notification and the function is non-nil.
	OnNewTickets func(hash Hash,
		height int64,
		stakeDiff int64,
		tickets []Hash)

	// OnStakeDifficulty is invoked when a block is connected to the longest
	// (best) chain and a new stake difficulty is calculated.  It will only
	// be invoked if a preceding call to NotifyStakeDifficulty has been
	// made to register for the notification and the function is non-nil.
	OnStakeDifficulty func(hash Hash,
		height int64,
		stakeDiff int64)

	// OnTxAccepted is invoked when a transaction is accepted into the
	// memory pool.  It will only be invoked if a preceding call to
	// NotifyNewTransactions with the verbose flag set to false has been
	// made to register for the notification and the function is non-nil.
	OnTxAccepted func(hash Hash, amount coin.Amount)

	// OnTxAccepted is invoked when a transaction is accepted into the
	// memory pool.  It will only be invoked if a preceding call to
	// NotifyNewTransactions with the verbose flag set to true has been
	// made to register for the notification and the function is non-nil.
	//OnTxAcceptedVerbose func(txDetails *dcrjson.TxRawResult)

	// OnNodeConnected is invoked when a wallet connects or disconnects from
	// node.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnNodeConnected func(connected bool)

	// OnAccountBalance is invoked with account balance updates.
	//
	// This will only be available when speaking to a wallet server
	// such as dcrwallet.
	OnAccountBalance func(account string, balance coin.Amount, confirmed bool)

	// OnWalletLockState is invoked when a wallet is locked or unlocked.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnWalletLockState func(locked bool)

	// OnTicketsPurchased is invoked when a wallet purchases an SStx.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnTicketsPurchased func(TxHash Hash, amount coin.Amount)

	// OnVotesCreated is invoked when a wallet generates an SSGen.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnVotesCreated func(txHash Hash,
		blockHash Hash,
		height int32,
		sstxIn Hash,
		voteBits uint16)

	// OnRevocationsCreated is invoked when a wallet generates an SSRtx.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnRevocationsCreated func(txHash Hash,
		sstxIn Hash)

	// OnUnknownNotification is invoked when an unrecognized notification
	// is received.  This typically means the notification handling code
	// for this package needs to be updated for a new notification type or
	// the caller is using a custom notification this package does not know
	// about.
	OnUnknownNotification func(method string, params []json.RawMessage)
}
