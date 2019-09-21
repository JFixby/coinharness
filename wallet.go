package coinharness

import "fmt"

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

	ListUnspent() ([]*Unspent, error)

	// ConfirmedBalance returns wallet balance
	GetBalance(accountName string) (*GetBalanceResult, error)

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
	ActiveNet     Network
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
	Outputs         []*OutputTx
	FeeRate         CoinsAmount
	Change          bool
	TxVersion       int32
	PayToAddrScript func(Address) ([]byte, error)
	TxSerializeSize func(*CreatedTransactionTx) int
	Account         string
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

// CreateTransaction returns a fully signed transaction paying to the specified
// outputs while observing the desired fee rate. The passed fee rate should be
// expressed in satoshis-per-byte. The transaction being created can optionally
// include a change output indicated by the Change boolean.
func CreateTransaction(wallet Wallet, args *CreateTransactionArgs) (*CreatedTransactionTx, error) {
	unspent, err := wallet.ListUnspent()
	if err != nil {
		return nil, err
	}

	tx := &CreatedTransactionTx{}

	// Tally up the total amount to be sent in order to perform coin
	// selection shortly below.
	outputAmt := CoinsAmount{0}
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
	wallet Wallet,
	account string,
	unspent []*Unspent,
	tx *CreatedTransactionTx,
	amt CoinsAmount,
	feeRate CoinsAmount,
	PayToAddrScript func(Address) ([]byte, error),
	TxSerializeSize func(*CreatedTransactionTx) int,
) error {
	const (
		// spendSize is the largest number of bytes of a sigScript
		// which spends a p2pkh output: OP_DATA_73 <sig> OP_DATA_33 <pubkey>
		spendSize = 1 + 73 + 1 + 33
	)

	amtSelected := CoinsAmount{0}
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
		txIn := &InputTx{
			PreviousOutPoint: OutPoint{
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
		reqFee := CoinsAmount{int64(txSize) * feeRate.AtomsValue}
		if amtSelected.AtomsValue-reqFee.AtomsValue < amt.AtomsValue {
			continue
		}

		// If we have any change left over, then add an additional
		// output to the transaction reserved for change.
		changeVal := CoinsAmount{amtSelected.AtomsValue - amt.AtomsValue - reqFee.AtomsValue}
		if changeVal.AtomsValue > 0 {
			addr, err := wallet.GetNewAddress(account)
			if err != nil {
				return err
			}
			pkScript, err := PayToAddrScript(addr)
			if err != nil {
				return err
			}
			changeOutput := &OutputTx{
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
