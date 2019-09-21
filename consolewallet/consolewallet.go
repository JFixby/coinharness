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

	AppDir    string
	ActiveNet coinharness.Network

	NodeRPCHost string
	NodeRPCPort int
	NodeUser    string
	NodePass    string

	WalletRPCHost string
	WalletRPCPort int
	WalletUser    string
	WalletPass    string
}

func NewConsoleWallet(args *NewConsoleWalletArgs) *ConsoleWallet {
	pin.AssertNotNil("WalletExecutablePathProvider", args.WalletExecutablePathProvider)
	pin.AssertNotNil("ActiveNet", args.ActiveNet)
	pin.AssertNotNil("ClientFac", args.ClientFac)

	pin.AssertNotNil("args.NodeRPCHost", args.NodeRPCHost)
	pin.AssertNotEmpty("args.NodeRPCHost", args.NodeRPCHost)

	pin.AssertNotNil("args.WalletRPCHost", args.WalletRPCHost)
	pin.AssertNotEmpty("args.WalletRPCHost", args.WalletRPCHost)

	pin.AssertTrue("args.NodeRPCPort", args.NodeRPCPort != 0)
	pin.AssertTrue("args.WalletRPCPort", args.WalletRPCPort != 0)

	Wallet := &ConsoleWallet{
		nodeRPCListener:              net.JoinHostPort(args.NodeRPCHost, strconv.Itoa(args.NodeRPCPort)),
		walletRpcListener:            net.JoinHostPort(args.WalletRPCHost, strconv.Itoa(args.WalletRPCPort)),
		WalletRpcUser:                args.WalletUser,
		WalletRpcPass:                args.WalletPass,
		NodeRpcUser:                  args.NodeUser,
		NodeRpcPass:                  args.NodePass,
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
// Implements harness.Testwallet.
type ConsoleWallet struct {
	// WalletExecutablePathProvider returns path to the dcrd executable
	WalletExecutablePathProvider commandline.ExecutablePathProvider

	NodeRpcUser string
	NodeRpcPass string

	WalletRpcUser string
	WalletRpcPass string

	nodeRPCListener   string
	walletRpcListener string
	appDir            string
	debugLevel        string
	endpoint          string

	externalProcess commandline.ExternalProcess

	rPCClient *coinharness.RPCConnection

	network coinharness.Network

	ConsoleCommandCook ConsoleCommandCook
}

type ConsoleCommandParams struct {
	ExtraArguments map[string]interface{}
	NodeRpcUser    string
	NodeRpcPass    string
	WalletRpcUser  string
	WalletRpcPass  string
	RpcConnect     string
	RpcListen      string
	AppDir         string
	DebugLevel     string
	CertFile       string
	NodeCertFile   string
	KeyFile        string
	Network        coinharness.Network
}

type ConsoleCommandCook interface {
	CookArguments(par *ConsoleCommandParams) map[string]interface{}
}

// RPCConnectionConfig produces a new connection config instance for RPC client
func (wallet *ConsoleWallet) RPCConnectionConfig() coinharness.RPCConnectionConfig {
	return coinharness.RPCConnectionConfig{
		Host:            wallet.walletRpcListener,
		Endpoint:        wallet.endpoint,
		User:            wallet.WalletRpcUser,
		Pass:            wallet.WalletRpcPass,
		CertificateFile: wallet.CertFile(),
	}
}

// FullConsoleCommand returns the full console command used to
// launch external process of the Wallet
func (wallet *ConsoleWallet) FullConsoleCommand() string {
	return wallet.externalProcess.FullConsoleCommand()
}

// RPCClient returns Wallet RPCConnection
func (wallet *ConsoleWallet) RPCClient() *coinharness.RPCConnection {
	return wallet.rPCClient
}

// CertFile returns file path of the .cert-file of the Wallet
func (wallet *ConsoleWallet) CertFile() string {
	return filepath.Join(wallet.appDir, "rpc.cert")
}

// KeyFile returns file path of the .key-file of the Wallet
func (wallet *ConsoleWallet) KeyFile() string {
	return filepath.Join(wallet.appDir, "rpc.key")
}

// Network returns current network of the Wallet
func (wallet *ConsoleWallet) Network() coinharness.Network {
	return wallet.network
}

// IsRunning returns true if ConsoleWallet is running external dcrd process
func (wallet *ConsoleWallet) IsRunning() bool {
	return wallet.externalProcess.IsRunning()
}

// Start Wallet process. Deploys working dir, launches dcrd using command-line,
// connects RPC client to the wallet.
func (wallet *ConsoleWallet) Start(args *coinharness.TestWalletStartArgs) error {
	if wallet.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("ConsoleWallet is already running"))
	}
	fmt.Println("Start Wallet process...")
	pin.MakeDirs(wallet.appDir)

	exec := wallet.WalletExecutablePathProvider.Executable()
	wallet.externalProcess.CommandName = exec

	consoleCommandParams := &ConsoleCommandParams{
		ExtraArguments: args.ExtraArguments,
		NodeRpcUser:    wallet.NodeRpcUser,
		NodeRpcPass:    wallet.NodeRpcPass,
		WalletRpcUser:  wallet.WalletRpcUser,
		WalletRpcPass:  wallet.WalletRpcPass,
		RpcConnect:     wallet.nodeRPCListener,
		RpcListen:      wallet.walletRpcListener,
		AppDir:         wallet.appDir,
		DebugLevel:     wallet.debugLevel,
		CertFile:       wallet.CertFile(),
		NodeCertFile:   args.NodeRPCCertFile,
		KeyFile:        wallet.KeyFile(),
		Network:        wallet.network,
	}

	wallet.externalProcess.Arguments = commandline.ArgumentsToStringArray(
		wallet.ConsoleCommandCook.CookArguments(consoleCommandParams),
	)
	wallet.externalProcess.Launch(args.DebugOutput)
	// Wallet RPC instance will create a cert file when it is ready for incoming calls
	pin.WaitForFile(wallet.CertFile(), 15)

	fmt.Println("Connect to Wallet RPC...")
	cfg := wallet.RPCConnectionConfig()
	wallet.rPCClient.Connect(cfg, nil)
	fmt.Println("Wallet RPC client connected.")

	return nil
}

// Stop interrupts the running Wallet process.
// Disconnects RPC client from the Wallet, removes cert-files produced by the dcrd,
// stops dcrd process.
func (wallet *ConsoleWallet) Stop() {
	if !wallet.IsRunning() {
		pin.ReportTestSetupMalfunction(fmt.Errorf("Wallet is not running"))
	}

	if wallet.rPCClient.IsConnected() {
		fmt.Println("Disconnect from Wallet RPC...")
		wallet.rPCClient.Disconnect()
	}

	fmt.Println("Stop Wallet process...")
	err := wallet.externalProcess.Stop()
	pin.CheckTestSetupMalfunction(err)

	// Delete files, RPC servers will recreate them on the next launch sequence
	pin.DeleteFile(wallet.CertFile())
	pin.DeleteFile(wallet.KeyFile())

}

// Dispose simply stops the Wallet process if running
func (wallet *ConsoleWallet) Dispose() error {
	if wallet.IsRunning() {
		wallet.Stop()
	}
	return nil
}

func (wallet *ConsoleWallet) NewAddress(arg *coinharness.NewAddressArgs) (coinharness.Address, error) {
	if arg == nil {
		arg = &coinharness.NewAddressArgs{}
		arg.Account = "default"
	}
	return wallet. //
			RPCClient().  //
			Connection(). //
			GetNewAddress(arg.Account)
}

func (wallet *ConsoleWallet) SendOutputs(args *coinharness.SendOutputsArgs) (coinharness.Hash, error) {
	panic("")
}

func (wallet *ConsoleWallet) Sync(desiredHeight int64) int64 {
	attempt := 0
	maxAttempt := 10
	for wallet.SyncedHeight() < desiredHeight {
		pin.Sleep(1000)
		count := wallet.SyncedHeight()
		fmt.Println("   sync to: " + strconv.FormatInt(count, 10))
		attempt++
		if maxAttempt <= attempt {

		}
	}
	return wallet.SyncedHeight()
}

func (wallet *ConsoleWallet) SyncedHeight() int64 {
	rpcClient := wallet.rPCClient.Connection()
	//h, err := rpcClient.GetBlockCount()
	_, h, err := rpcClient.GetBestBlock()
	pin.CheckTestSetupMalfunction(err)
	return h
}

//func (wallet *ConsoleWallet) UnlockOutputs(inputs []coinharness.InputTx) {
//	wallet.rPCClient.Connection().UnlockOutputs(inputs)
//}

func (wallet *ConsoleWallet) CreateNewAccount(accountName string) error {
	return wallet.rPCClient.Connection().CreateNewAccount(accountName)
}

func (wallet *ConsoleWallet) GetNewAddress(accountName string) (coinharness.Address, error) {
	return wallet.rPCClient.Connection().GetNewAddress(accountName)
}

func (wallet *ConsoleWallet) ValidateAddress(address coinharness.Address) (*coinharness.ValidateAddressResult, error) {
	return wallet.rPCClient.Connection().ValidateAddress(address)
}

func (wallet *ConsoleWallet) CreateTransaction(args *coinharness.CreateTransactionArgs) (*coinharness.CreatedTransactionTx, error) {
	unspent, err := wallet.rPCClient.Connection().ListUnspent()
	if err != nil {
		return nil, err
	}

	tx := &coinharness.CreatedTransactionTx{}

	// Tally up the total amount to be sent in order to perform coin
	// selection shortly below.
	outputAmt := coinharness.CoinsAmount{0}
	for _, output := range args.Outputs {
		outputAmt.AtomsValue += output.Amount.AtomsValue
		tx.TxOut = append(tx.TxOut, output)
	}

	// Attempt to fund the transaction with spendable utxos.
	if err := fundTx(
		wallet,
		args.Account,
		unspent,
		tx,
		outputAmt,
		args.FeeRate,
		args.PayToAddrScript,
		args.TxSerializeSize,
	); err != nil {
		return nil, err
	}

	return tx, nil
}

// fundTx attempts to fund a transaction sending amt coins.  The coins are
// selected such that the final amount spent pays enough fees as dictated by
// the passed fee rate.  The passed fee rate should be expressed in
// atoms-per-byte.
//
// NOTE: The InMemoryWallet's mutex must be held when this function is called.
func fundTx(
	wallet *ConsoleWallet,
	account string,
	unspent []*coinharness.Unspent,
	tx *coinharness.CreatedTransactionTx,
	amt coinharness.CoinsAmount,
	feeRate coinharness.CoinsAmount,
	PayToAddrScript func(coinharness.Address) ([]byte, error),
	TxSerializeSize func(*coinharness.CreatedTransactionTx) int,
) error {
	const (
		// spendSize is the largest number of bytes of a sigScript
		// which spends a p2pkh output: OP_DATA_73 <sig> OP_DATA_33 <pubkey>
		spendSize = 1 + 73 + 1 + 33
	)

	amtSelected := coinharness.CoinsAmount{0}
	//txSize := int64(0)
	for _, output := range unspent {
		// Skip any outputs that are still currently immature or are
		// currently locked.
		if !output.Spendable {
			continue
		}
		if output.Account != account {
			continue
		}

		amtSelected.AtomsValue += output.Amount.AtomsValue

		// Add the selected output to the transaction, updating the
		// current tx size while accounting for the size of the future
		// sigScript.
		txIn := &coinharness.InputTx{
			PreviousOutPoint: coinharness.OutPoint{
				Tree: output.Tree,
			},
			Amount: output.Amount.Copy(),
		}
		tx.TxIn = append(tx.TxIn, txIn)

		txSize := TxSerializeSize(tx) + spendSize*len(tx.TxIn)

		// Calculate the fee required for the txn at this point
		// observing the specified fee rate. If we don't have enough
		// coins from he current amount selected to pay the fee, then
		// continue to grab more coins.
		reqFee := coinharness.CoinsAmount{int64(txSize) * feeRate.AtomsValue}
		if amtSelected.AtomsValue-reqFee.AtomsValue < amt.AtomsValue {
			continue
		}

		// If we have any change left over, then add an additional
		// output to the transaction reserved for change.
		changeVal := coinharness.CoinsAmount{amtSelected.AtomsValue - amt.AtomsValue - reqFee.AtomsValue}
		if changeVal.AtomsValue > 0 {
			addr, err := wallet.GetNewAddress(account)
			if err != nil {
				return err
			}
			pkScript, err := PayToAddrScript(addr)
			if err != nil {
				return err
			}
			changeOutput := &coinharness.OutputTx{
				Amount:   changeVal,
				PkScript: pkScript,
			}
			tx.TxOut = append(tx.TxOut, changeOutput)
		}
		return nil
	}

	// If we've reached this point, then coin selection failed due to an
	// insufficient amount of coins.
	return fmt.Errorf("not enough funds for coin selection")
}

func (wallet *ConsoleWallet) GetBalance(accountName string) (*coinharness.GetBalanceResult, error) {
	return wallet.rPCClient.Connection().GetBalance(accountName)
}

func (wallet *ConsoleWallet) UnlockOutputs(inputs []coinharness.InputTx) error {
	return fmt.Errorf("UnlockOutputs method is not supported")
}

func (wallet *ConsoleWallet) WalletUnlock(walletPassphrase string, timeout int64) error {
	return wallet.rPCClient.Connection().WalletUnlock(walletPassphrase, timeout)
}

func (wallet *ConsoleWallet) WalletLock() error {
	return wallet.rPCClient.Connection().WalletLock()
}

func (wallet *ConsoleWallet) WalletInfo() (*coinharness.WalletInfoResult, error) {
	return wallet.rPCClient.Connection().WalletInfo()
}
