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
	GetRawMempool(command interface{}) ([]Hash, error)
	AddNode(arguments *AddNodeArguments) error
	Internal() interface{}
	Generate(blocks uint32) ([]Hash, error)
	SendRawTransaction(tx CreatedTransactionTx, b bool) (Hash, error)
	GetNewAddress(accountName string) (Address, error)
	GetBuildVersion() (BuildVersion, error)
	GetBestBlock() (Hash, int64, error)
	ValidateAddress(address Address) (*ValidateAddressResult, error)
	CreateNewAccount(accountName string) error
	GetBalance(accountName string) (*GetBalanceResult, error)
	ListUnspent() ([]*Unspent, error)
	//UnlockOutputs(inputs []InputTx)
	//CreateTransaction(args *CreateTransactionArgs) (CreatedTransactionTx, error)

	WalletUnlock(walletPassphrase string, timeout int64) error
	WalletLock() error
	WalletInfo() (*WalletInfoResult, error)
	GetBlock(hash Hash) (Block, error)
	SubmitBlock(block Block) error
	LoadTxFilter(b bool, addresses []Address) error
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
	Amount        CoinsAmount
	Confirmations int64
	Spendable     bool
}

type Block interface {
}

type PeerInfo struct {
	Addr string
}

type BuildVersion interface {
	VersionString() string
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
