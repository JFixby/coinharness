package coinharness

import (
	"bytes"
	"encoding/json"
	"github.com/jfixby/pin"
	"sync"
	"time"
)

// InMemoryWallet is a simple in-memory wallet whose purpose is to provide basic
// wallet functionality to the harness. The wallet uses a hard-coded HD key
// hierarchy which promotes reproducibility between harness test runs.
// Implements harness.TestWallet.
type InMemoryWallet struct {
	CoinbaseKey  CoinbaseKey //*secp256k1.PrivateKey
	CoinbaseAddr Address

	// HdRoot is the root master private key for the wallet.
	HdRoot ExtendedKey //*hdkeychain.ExtendedKey

	// HdIndex is the next available key index offset from the HdRoot.
	HdIndex uint32

	// currentHeight is the latest height the wallet is known to be synced
	// to.
	currentHeight int64

	// addrs tracks all addresses belonging to the wallet. The addresses
	// are indexed by their keypath from the HdRoot.
	Addrs map[uint32]Address

	// Utxos is the set of Utxos spendable by the wallet.
	Utxos map[OutPoint]*Utxo

	// ReorgJournal is a map storing an undo entry for each new block
	// received. Once a block is disconnected, the undo entry for the
	// particular height is evaluated, thereby rewinding the effect of the
	// disconnected block on the wallet's set of spendable Utxos.
	ReorgJournal map[int64]*UndoEntry

	chainUpdates []*chainUpdate

	// chainUpdateSignal is a wallet event queue
	ChainUpdateSignal chan string

	chainMtx sync.Mutex

	Net Network

	nodeRPC RPCClient

	sync.RWMutex
	RPCClientFactory RPCClientFactory

	NewTxFromBytes      func(txBytes []byte) (*Tx, error) //dcrutil.NewTxFromBytes(txBytes)
	IsCoinBaseTx        func(*MessageTx) bool             //blockchain.IsCoinBaseTx(mtx)
	PrivateKeyKeyToAddr func(key PrivateKey, net Network) (Address, error)
	ReadBlockHeader     func(header []byte) BlockHeader
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
	OnTxAccepted func(hash Hash, amount CoinsAmount)

	// OnTxAccepted is invoked when a transaction is accepted into the
	// memory pool.  It will only be invoked if a preceding call to
	// NotifyNewTransactions with the verbose flag set to true has been
	// made to register for the notification and the function is non-nil.
	//OnTxAcceptedVerbose func(txDetails *dcrjson.TxRawResult)

	// OnDcrdConnected is invoked when a wallet connects or disconnects from
	// dcrd.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnDcrdConnected func(connected bool)

	// OnAccountBalance is invoked with account balance updates.
	//
	// This will only be available when speaking to a wallet server
	// such as dcrwallet.
	OnAccountBalance func(account string, balance CoinsAmount, confirmed bool)

	// OnWalletLockState is invoked when a wallet is locked or unlocked.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnWalletLockState func(locked bool)

	// OnTicketsPurchased is invoked when a wallet purchases an SStx.
	//
	// This will only be available when client is connected to a wallet
	// server such as dcrwallet.
	OnTicketsPurchased func(TxHash Hash, amount CoinsAmount)

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

const chainUpdateSignal = "chainUpdateSignal"
const stopSignal = "stopSignal"

// chainUpdate encapsulates an update to the current main chain. This struct is
// used to sync up the InMemoryWallet each time a new block is connected to the main
// chain.
type chainUpdate struct {
	blockHeight  int64
	filteredTxns []*Tx
}

// UndoEntry is functionally the opposite of a chainUpdate. An UndoEntry is
// created for each new block received, then stored in a log in order to
// properly handle block re-orgs.
type UndoEntry struct {
	utxosDestroyed map[OutPoint]*Utxo
	utxosCreated   []OutPoint
}

// Utxo represents an unspent output spendable by the InMemoryWallet. The maturity
// height of the transaction is recorded in order to properly observe the
// maturity period of direct coinbase outputs.
type Utxo struct {
	pkScript       []byte
	value          CoinsAmount
	maturityHeight int64
	keyIndex       uint32
	isLocked       bool
}

// isMature returns true if the target Utxo is considered "mature" at the
// passed block height. Otherwise, false is returned.
func (u *Utxo) isMature(height int64) bool {
	return height >= u.maturityHeight
}
func (wallet *InMemoryWallet) ListAccounts() (map[string]CoinsAmount, error) {

	l, err := wallet.GetBalance()
	if err != nil {
		return nil, err
	}

	r := make(map[string]CoinsAmount)
	for k, v := range l.Balances {
		r[k] = v.Spendable.Copy()
	}
	return r, nil
}

func (wallet *InMemoryWallet) SendFrom(account string, address Address, amount CoinsAmount) error {
	panic("implement me")
}

// Network returns current network of the wallet
func (wallet *InMemoryWallet) Network() Network {
	return wallet.Net
}

// Start wallet process
func (wallet *InMemoryWallet) Start(args *TestWalletStartArgs) error {
	handlers := &NotificationHandlers{}

	// If a handler for the OnBlockConnected/OnBlockDisconnected callback
	// has already been set, then we create a wrapper callback which
	// executes both the currently registered callback, and the mem
	// wallet's callback.
	if handlers.OnBlockConnected != nil {
		obc := handlers.OnBlockConnected
		handlers.OnBlockConnected = func(header []byte, filteredTxns [][]byte) {
			wallet.IngestBlock(header, filteredTxns)
			obc(header, filteredTxns)
		}
	} else {
		// Otherwise, we can claim the callback ourselves.
		handlers.OnBlockConnected = wallet.IngestBlock
	}
	if handlers.OnBlockDisconnected != nil {
		obd := handlers.OnBlockDisconnected
		handlers.OnBlockDisconnected = func(header []byte) {
			wallet.UnwindBlock(header)
			obd(header)
		}
	} else {
		handlers.OnBlockDisconnected = wallet.UnwindBlock
	}

	wallet.nodeRPC = NewRPCConnection(wallet.RPCClientFactory, args.NodeRPCConfig, 5, handlers)
	pin.AssertNotNil("nodeRPC", wallet.nodeRPC)

	// Filter transactions that pay to the coinbase associated with the
	// wallet.
	wallet.updateTxFilter()

	// Ensure dcrd properly dispatches our registered call-back for each new
	// block. Otherwise, the InMemoryWallet won't function properly.
	err := wallet.nodeRPC.NotifyBlocks()
	pin.CheckTestSetupMalfunction(err)

	go wallet.chainSyncer()
	return nil
}

func (wallet *InMemoryWallet) updateTxFilter() {
	filterAddrs := []Address{}
	for _, v := range wallet.Addrs {
		filterAddrs = append(filterAddrs, v)
	}
	err := wallet.nodeRPC.LoadTxFilter(true, filterAddrs)
	pin.CheckTestSetupMalfunction(err)
}

// Stop wallet process gently, by sending stopSignal to the wallet event queue
func (wallet *InMemoryWallet) Stop() {
	go func() {
		wallet.ChainUpdateSignal <- stopSignal
	}()
	wallet.nodeRPC.Disconnect()
	wallet.nodeRPC = nil
}

// Sync block until the wallet has fully synced up to the tip of the main
// chain.
func (wallet *InMemoryWallet) Sync(desiredHeight int64) int64 {
	ticker := time.NewTicker(time.Millisecond * 100)
	for range ticker.C {
		walletHeight := wallet.SyncedHeight()
		if walletHeight >= desiredHeight {
			break
		}
	}
	return wallet.SyncedHeight()
}

// Dispose is no needed for InMemoryWallet
func (wallet *InMemoryWallet) Dispose() error {
	return nil
}

// SyncedHeight returns the height the wallet is known to be synced to.
//
// This function is safe for concurrent access.
func (wallet *InMemoryWallet) SyncedHeight() int64 {
	wallet.RLock()
	defer wallet.RUnlock()
	return wallet.currentHeight
}

// IngestBlock is a call-back which is to be triggered each time a new block is
// connected to the main chain. Ingesting a block updates the wallet's internal
// Utxo state based on the outputs created and destroyed within each block.
func (m *InMemoryWallet) IngestBlock(headerBytes []byte, filteredTxns [][]byte) {
	//var hdr wire.BlockHeader
	//if err := hdr.FromBytes(header); err != nil {
	//	panic(err)
	//}
	//height := int64(hdr.Height)
	header := m.ReadBlockHeader(headerBytes)
	height := header.Height()

	txns := make([]*Tx, 0, len(filteredTxns))
	for _, txBytes := range filteredTxns {
		tx, err := m.NewTxFromBytes(txBytes)

		if err != nil {
			panic(err)
		}
		txns = append(txns, tx)
	}

	// Append this new chain update to the end of the queue of new chain
	// updates.
	m.chainMtx.Lock()
	m.chainUpdates = append(m.chainUpdates, &chainUpdate{height, txns})
	m.chainMtx.Unlock()

	// Launch a goroutine to signal the chainSyncer that a new update is
	// available. We do this in a new goroutine in order to avoid blocking
	// the main loop of the rpc client.
	go func() {
		m.ChainUpdateSignal <- chainUpdateSignal
	}()
}

//// ingestBlock updates the wallet's internal Utxo state based on the outputs
//// created and destroyed within each block.
//func (wallet *InMemoryWallet) ingestBlock(update *chainUpdate) {
//	// Update the latest synced height, then process each filtered
//	// transaction in the block creating and destroying Utxos within
//	// the wallet as a result.
//	wallet.currentHeight = update.blockHeight
//	undo := &UndoEntry{
//		utxosDestroyed: make(map[coinharness.OutPoint]*Utxo),
//	}
//	for _, tx := range update.filteredTxns {
//		mtx := tx.MsgTx()
//		isCoinbase := blockchain.IsCoinBaseTx(mtx)
//		txHash := mtx.TxHash()
//		wallet.evalOutputs(mtx.TxOut, &txHash, isCoinbase, undo)
//		wallet.evalInputs(mtx.TxIn, undo)
//	}
//
//	// Finally, record the undo entry for this block so we can
//	// properly update our internal state in response to the block
//	// being re-org'd from the main chain.
//	wallet.ReorgJournal[update.blockHeight] = undo
//}

// chainSyncer is a goroutine dedicated to processing new blocks in order to
// keep the wallet's Utxo state up to date.
//
// NOTE: This MUST be run as a goroutine.
func (wallet *InMemoryWallet) chainSyncer() {
	var update *chainUpdate

	for s := range wallet.ChainUpdateSignal {
		if s == stopSignal {
			break
		}
		// A new update is available, so pop the new chain update from
		// the front of the update queue.
		wallet.chainMtx.Lock()
		update = wallet.chainUpdates[0]
		wallet.chainUpdates[0] = nil // Set to nil to prevent GC leak.
		wallet.chainUpdates = wallet.chainUpdates[1:]
		wallet.chainMtx.Unlock()

		// Update the latest synced height, then process each filtered
		// transaction in the block creating and destroying Utxos within
		// the wallet as a result.
		wallet.Lock()
		wallet.currentHeight = update.blockHeight
		undo := &UndoEntry{
			utxosDestroyed: make(map[OutPoint]*Utxo),
		}
		for _, tx := range update.filteredTxns {
			mtx := tx.MsgTx
			isCoinbase := wallet.IsCoinBaseTx(mtx)
			txHash := mtx.TxHash
			wallet.evalOutputs(mtx.TxOut, txHash, isCoinbase, undo)
			wallet.evalInputs(mtx.TxIn, undo)
		}

		// Finally, record the undo entry for this block so we can
		// properly update our internal state in response to the block
		// being re-org'd from the main chain.
		wallet.ReorgJournal[update.blockHeight] = undo
		wallet.Unlock()
	}
}

// evalOutputs evaluates each of the passed outputs, creating a new matching
// Utxo within the wallet if we're able to spend the output.
func (wallet *InMemoryWallet) evalOutputs(outputs []*TxOut, txHash Hash, isCoinbase bool, undo *UndoEntry) {
	for i, output := range outputs {
		pkScript := output.PkScript

		// Scan all the addresses we currently control to see if the
		// output is paying to us.
		for keyIndex, addr := range wallet.Addrs {
			pkHash := addr.ScriptAddress()
			if !bytes.Contains(pkScript, pkHash) {
				continue
			}

			// If this is a coinbase output, then we mark the
			// maturity height at the proper block height in the
			// future.
			var maturityHeight int64
			if isCoinbase {
				maturityHeight = wallet.currentHeight + int64(wallet.Net.CoinbaseMaturity())
			}

			op := OutPoint{Hash: txHash, Index: uint32(i)}
			wallet.Utxos[op] = &Utxo{
				value:          output.Value.Copy(),
				keyIndex:       keyIndex,
				maturityHeight: maturityHeight,
				pkScript:       pkScript,
			}
			undo.utxosCreated = append(undo.utxosCreated, op)
		}
	}
}

// evalInputs scans all the passed inputs, destroying any Utxos within the
// wallet which are spent by an input.
func (wallet *InMemoryWallet) evalInputs(inputs []*TxIn, undo *UndoEntry) {
	for _, txIn := range inputs {
		op := txIn.PreviousOutPoint
		oldUtxo, ok := wallet.Utxos[op]
		if !ok {
			continue
		}

		undo.utxosDestroyed[op] = oldUtxo
		delete(wallet.Utxos, op)
	}
}

// UnwindBlock is a call-back which is to be executed each time a block is
// disconnected from the main chain. Unwinding a block undoes the effect that a
// particular block had on the wallet's internal Utxo state.
func (m *InMemoryWallet) UnwindBlock(headerBytes []byte) {
	header := m.ReadBlockHeader(headerBytes)
	//var hdr wire.BlockHeader
	//if err := hdr.FromBytes(header); err != nil {
	//	panic(err)
	//}
	//height := int64(hdr.Height)
	height := header.Height()
	m.Lock()
	defer m.Unlock()

	undo := m.ReorgJournal[height]

	for _, utxo := range undo.utxosCreated {
		delete(m.Utxos, utxo)
	}

	for outPoint, utxo := range undo.utxosDestroyed {
		m.Utxos[outPoint] = utxo
	}

	delete(m.ReorgJournal, height)
}

// unwindBlock undoes the effect that a particular block had on the wallet's
// internal Utxo state.
func (wallet *InMemoryWallet) unwindBlock(update *chainUpdate) {
	undo := wallet.ReorgJournal[update.blockHeight]

	for _, utxo := range undo.utxosCreated {
		delete(wallet.Utxos, utxo)
	}

	for outPoint, utxo := range undo.utxosDestroyed {
		wallet.Utxos[outPoint] = utxo
	}

	delete(wallet.ReorgJournal, update.blockHeight)
}

// newAddress returns a new address from the wallet's hd key chain.  It also
// loads the address into the RPC client's transaction filter to ensure any
// transactions that involve it are delivered via the notifications.
func (wallet *InMemoryWallet) newAddress() (Address, error) {
	index := wallet.HdIndex

	childKey, err := wallet.HdRoot.Child(index)
	if err != nil {
		return nil, err
	}
	privKey, err := childKey.PrivateKey()
	if err != nil {
		return nil, err
	}

	addr, err := wallet.PrivateKeyKeyToAddr(privKey, wallet.Net)
	if err != nil {
		return nil, err
	}

	err = wallet.nodeRPC.LoadTxFilter(false, []Address{addr})
	if err != nil {
		return nil, err
	}

	wallet.Addrs[index] = addr

	wallet.HdIndex++

	return addr, nil
}

// NewAddress returns a fresh address spendable by the wallet.
//
// This function is safe for concurrent access.
func (wallet *InMemoryWallet) NewAddress(account string) (Address, error) {
	wallet.Lock()
	defer wallet.Unlock()

	add, err := wallet.newAddress()

	if err != nil {
		return nil, err
	}

	return add, nil
}

func (wallet *InMemoryWallet) ListUnspent() (result []*Unspent, err error) {
	wallet.Lock()
	defer wallet.Unlock()

	panic("not implemented")
}

// UnlockOutputs unlocks any outputs which were previously locked due to
// being selected to fund a transaction via the CreateTransaction method.
//
// This function is safe for concurrent access.
func (wallet *InMemoryWallet) UnlockOutputs(inputs []TxIn) error {
	wallet.Lock()
	defer wallet.Unlock()

	for _, input := range inputs {
		utxo, ok := wallet.Utxos[input.PreviousOutPoint]
		if !ok {
			continue
		}

		utxo.isLocked = false
	}

	return nil
}

// GetBalance returns the confirmed balance of the wallet.
//
// This function is safe for concurrent access.
func (wallet *InMemoryWallet) GetBalance() (*GetBalanceResult, error) {
	wallet.RLock()
	defer wallet.RUnlock()
	result := &GetBalanceResult{}
	//result.BlockHash
	b := GetAccountBalanceResult{}
	result.Balances[DefaultAccountName] = b

	balance := CoinsAmount{0}
	for _, utxo := range wallet.Utxos {
		// Prevent any immature or locked outputs from contributing to
		// the wallet's total confirmed balance.
		if !utxo.isMature(wallet.currentHeight) || utxo.isLocked {
			continue
		}

		balance.AtomsValue += utxo.value.AtomsValue
	}

	b.Spendable = balance
	b.AccountName = DefaultAccountName
	return result, nil
}

func (wallet *InMemoryWallet) RPCClient() *RPCConnection {
	panic("Method not supported")
}

func (wallet *InMemoryWallet) CreateNewAccount(accountName string) error {
	panic("")
}

func (wallet *InMemoryWallet) GetNewAddress(accountName string) (Address, error) {
	panic("")
}
func (wallet *InMemoryWallet) ValidateAddress(address Address) (*ValidateAddressResult, error) {
	panic("")
}

func (wallet *InMemoryWallet) WalletUnlock(password string, seconds int64) error {
	return nil
}
func (wallet *InMemoryWallet) WalletInfo() (*WalletInfoResult, error) {
	return &WalletInfoResult{
		Unlocked: true,
	}, nil
}
func (wallet *InMemoryWallet) WalletLock() error {
	return nil
}
