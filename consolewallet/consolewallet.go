package consolewallet

import (
	"fmt"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
	"net"
	"path/filepath"
	"strconv"
)

type NewConsoleWalletArgs struct {
	ClientFac                    coinharness.RPCClientFactory
	ConsoleCommandCook           ConsoleCommandCook
	WalletExecutablePathProvider commandline.ExecutablePathProvider
	RpcUser                      string
	RpcPass                      string
	AppDir                       string
	ActiveNet                    coinharness.Network
	WalletRPCHost                string
	WalletRPCPort                int
}

func NewConsoleWallet(args *NewConsoleWalletArgs) *ConsoleWallet {
	pin.AssertNotNil("WalletExecutablePathProvider", args.WalletExecutablePathProvider)
	pin.AssertNotNil("ActiveNet", args.ActiveNet)
	pin.AssertNotNil("ClientFac", args.ClientFac)

	Wallet := &ConsoleWallet{
		rpcListen:                    net.JoinHostPort(args.WalletRPCHost, strconv.Itoa(args.WalletRPCPort)),
		rpcUser:                      args.RpcUser,
		rpcPass:                      args.RpcPass,
		appDir:                       args.AppDir,
		endpoint:                     "ws",
		rPCClient:                    &coinharness.RPCConnection{MaxConnRetries: 20, RPCClientFactory: args.ClientFac},
		WalletExecutablePathProvider: args.WalletExecutablePathProvider,
		network:                      args.ActiveNet,
		ConsoleCommandCook:           args.ConsoleCommandCook,
	}
	return Wallet
}

// ConsoleWallet launches a new dcrd instance using command-line call.
// Implements harness.TestWallet.
type ConsoleWallet struct {
	// WalletExecutablePathProvider returns path to the dcrd executable
	WalletExecutablePathProvider commandline.ExecutablePathProvider

	rpcUser    string
	rpcPass    string
	rpcListen  string
	rpcConnect string
	profile    string
	debugLevel string
	appDir     string
	endpoint   string

	externalProcess commandline.ExternalProcess

	rPCClient *coinharness.RPCConnection

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
	AppDir         string
	DebugLevel     string
	Profile        string
	CertFile       string
	KeyFile        string
	Network        coinharness.Network
}

type ConsoleCommandCook interface {
	CookArguments(par *ConsoleCommandParams) map[string]interface{}
}

// RPCConnectionConfig produces a new connection config instance for RPC client
func (Wallet *ConsoleWallet) RPCConnectionConfig() coinharness.RPCConnectionConfig {
	return coinharness.RPCConnectionConfig{
		Host:            Wallet.rpcListen,
		Endpoint:        Wallet.endpoint,
		User:            Wallet.rpcUser,
		Pass:            Wallet.rpcPass,
		CertificateFile: Wallet.CertFile(),
	}
}

// FullConsoleCommand returns the full console command used to
// launch external process of the Wallet
func (Wallet *ConsoleWallet) FullConsoleCommand() string {
	return Wallet.externalProcess.FullConsoleCommand()
}

func (Wallet *ConsoleWallet) SetExtraArguments(WalletExtraArguments map[string]interface{}) {
	if Wallet.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("Unable to set parameter, ConsoleWallet is already running"))
	}

	Wallet.extraArgs = WalletExtraArguments
}

func (Wallet *ConsoleWallet) SetDebugWalletOutput(d bool) {
	if Wallet.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("Unable to set parameter, ConsoleWallet is already running"))
	}
	Wallet.debugOutput = d
}

// RPCClient returns Wallet RPCConnection
func (Wallet *ConsoleWallet) RPCClient() *coinharness.RPCConnection {
	return Wallet.rPCClient
}

// CertFile returns file path of the .cert-file of the Wallet
func (Wallet *ConsoleWallet) CertFile() string {
	return filepath.Join(Wallet.appDir, "rpc.cert")
}

// KeyFile returns file path of the .key-file of the Wallet
func (Wallet *ConsoleWallet) KeyFile() string {
	return filepath.Join(Wallet.appDir, "rpc.key")
}

// Network returns current network of the Wallet
func (Wallet *ConsoleWallet) Network() coinharness.Network {
	return Wallet.network
}

// IsRunning returns true if ConsoleWallet is running external dcrd process
func (Wallet *ConsoleWallet) IsRunning() bool {
	return Wallet.externalProcess.IsRunning()
}

// Start Wallet process. Deploys working dir, launches dcrd using command-line,
// connects RPC client to the Wallet.
func (Wallet *ConsoleWallet) Start(args *coinharness.TestWalletStartArgs) error {
	if Wallet.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("ConsoleWallet is already running"))
	}
	fmt.Println("Start Wallet process...")
	pin.MakeDirs(Wallet.appDir)

	exec := Wallet.WalletExecutablePathProvider.Executable()
	Wallet.externalProcess.CommandName = exec

	consoleCommandParams := &ConsoleCommandParams{
		ExtraArguments: Wallet.extraArgs,
		RpcUser:        Wallet.rpcUser,
		RpcPass:        Wallet.rpcPass,
		RpcConnect:     Wallet.rpcConnect,
		RpcListen:      Wallet.rpcListen,
		AppDir:         Wallet.appDir,
		DebugLevel:     Wallet.debugLevel,
		Profile:        Wallet.profile,
		CertFile:       Wallet.CertFile(),
		KeyFile:        Wallet.KeyFile(),
		Network:        Wallet.network,
	}

	Wallet.externalProcess.Arguments = commandline.ArgumentsToStringArray(
		Wallet.ConsoleCommandCook.CookArguments(consoleCommandParams),
	)
	Wallet.externalProcess.Launch(Wallet.debugOutput)
	// Wallet RPC instance will create a cert file when it is ready for incoming calls
	pin.WaitForFile(Wallet.CertFile(), 7)

	fmt.Println("Connect to Wallet RPC...")
	cfg := Wallet.RPCConnectionConfig()
	Wallet.rPCClient.Connect(cfg, nil)
	fmt.Println("Wallet RPC client connected.")

	panic("")
}

// Stop interrupts the running Wallet process.
// Disconnects RPC client from the Wallet, removes cert-files produced by the dcrd,
// stops dcrd process.
func (Wallet *ConsoleWallet) Stop() {
	if !Wallet.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("Wallet is not running"))
	}

	if Wallet.rPCClient.IsConnected() {
		fmt.Println("Disconnect from Wallet RPC...")
		Wallet.rPCClient.Disconnect()
	}

	fmt.Println("Stop Wallet process...")
	err := Wallet.externalProcess.Stop()
	pin.CheckTestSetupMalfunction(err)

	// Delete files, RPC servers will recreate them on the next launch sequence
	pin.DeleteFile(Wallet.CertFile())
	pin.DeleteFile(Wallet.KeyFile())

	panic("")
}

// Dispose simply stops the Wallet process if running
func (Wallet *ConsoleWallet) Dispose() error {
	if Wallet.IsRunning() {
		Wallet.Stop()
	}
	return nil
}

func (Wallet *ConsoleWallet) ConfirmedBalance() coinharness.CoinsAmount {
	panic("")
}

func (wallet *ConsoleWallet) CreateTransaction(args *coinharness.CreateTransactionArgs) (coinharness.CreatedTransactionTx, error) {
	panic("")
}

func (wallet *ConsoleWallet) NewAddress(_ *coinharness.NewAddressArgs) (coinharness.Address, error) {
	panic("")
}

func (wallet *ConsoleWallet) SendOutputs(args *coinharness.SendOutputsArgs) (coinharness.Hash, error) {
	panic("")
}

func (wallet *ConsoleWallet) Sync() {
	panic("")
}

func (wallet *ConsoleWallet) UnlockOutputs(inputs []coinharness.InputTx) {
	panic("")
}

func (wallet *ConsoleWallet) CreateNewAccount(accountName string) error {
	panic("")
}

func (wallet *ConsoleWallet) GetBalance(accountName string) (coinharness.CoinsAmount, error) {
	panic("")
}

func (wallet *ConsoleWallet) GetNewAddress(accountName string) (coinharness.Address, error) {
	panic("")
}

func (wallet *ConsoleWallet) ValidateAddress(address coinharness.Address) (*coinharness.ValidateAddressResult, error) {
	panic("")
}
