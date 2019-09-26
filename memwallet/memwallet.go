package memwallet

import (
	"bytes"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"sync"
	"time"
)

// InMemoryWallet is a simple in-memory wallet whose purpose is to provide basic
// wallet functionality to the harness. The wallet uses a hard-coded HD key
// hierarchy which promotes reproducibility between harness test runs.
// Implements harness.TestWallet.
type InMemoryWallet struct {
	coinbaseKey  coinharness.CoinbaseKey //*secp256k1.PrivateKey
	coinbaseAddr coinharness.Address

	// hdRoot is the root master private key for the wallet.
	hdRoot coinharness.ExtendedKey //*hdkeychain.ExtendedKey

	// hdIndex is the next available key index offset from the hdRoot.
	hdIndex uint32

	// currentHeight is the latest height the wallet is known to be synced
	// to.
	currentHeight int64

	// addrs tracks all addresses belonging to the wallet. The addresses
	// are indexed by their keypath from the hdRoot.
	addrs map[uint32]coinharness.Address

	// utxos is the set of utxos spendable by the wallet.
	utxos map[coinharness.OutPoint]*utxo

	// reorgJournal is a map storing an undo entry for each new block
	// received. Once a block is disconnected, the undo entry for the
	// particular height is evaluated, thereby rewinding the effect of the
	// disconnected block on the wallet's set of spendable utxos.
	reorgJournal map[int64]*undoEntry

	chainUpdates []*chainUpdate

	// chainUpdateSignal is a wallet event queue
	chainUpdateSignal chan string

	chainMtx sync.Mutex

	net coinharness.Network

	nodeRPC coinharness.RPCClient

	sync.RWMutex
	RPCClientFactory coinharness.RPCClientFactory

	NewTxFromBytes      func(txBytes []byte) (*coinharness.Tx, error) //dcrutil.NewTxFromBytes(txBytes)
	IsCoinBaseTx        func(*coinharness.MessageTx) (bool)           //blockchain.IsCoinBaseTx(mtx)
	PrivateKeyKeyToAddr func(key coinharness.PrivateKey, net coinharness.Network) (coinharness.Address, error)
	ReadBlockHeader     func(header []byte) coinharness.BlockHeader
}

func (wallet *InMemoryWallet) ListAccounts() (map[string]coinharness.CoinsAmount, error) {

	l, err := wallet.GetBalance()
	if err != nil {
		return nil, err
	}

	r := make(map[string]coinharness.CoinsAmount)
	for k, v := range l.Balances {
		r[k] = v.Spendable.Copy()
	}
	return r, nil
}

func (wallet *InMemoryWallet) SendFrom(account string, address coinharness.Address, amount coinharness.CoinsAmount) (error) {
	panic("implement me")
}

// Network returns current network of the wallet
func (wallet *InMemoryWallet) Network() coinharness.Network {
	return wallet.net
}

// Start wallet process
func (wallet *InMemoryWallet) Start(args *coinharness.TestWalletStartArgs) error {
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

	wallet.nodeRPC = coinharness.NewRPCConnection(wallet.RPCClientFactory, args.NodeRPCConfig, 5, handlers)
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
	filterAddrs := []coinharness.Address{}
	for _, v := range wallet.addrs {
		filterAddrs = append(filterAddrs, v)
	}
	err := wallet.nodeRPC.LoadTxFilter(true, filterAddrs)
	pin.CheckTestSetupMalfunction(err)
}

// Stop wallet process gently, by sending stopSignal to the wallet event queue
func (wallet *InMemoryWallet) Stop() {
	go func() {
		wallet.chainUpdateSignal <- stopSignal
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
// utxo state based on the outputs created and destroyed within each block.
func (m *InMemoryWallet) IngestBlock(headerBytes []byte, filteredTxns [][]byte) {
	//var hdr wire.BlockHeader
	//if err := hdr.FromBytes(header); err != nil {
	//	panic(err)
	//}
	//height := int64(hdr.Height)
	header := m.ReadBlockHeader(headerBytes)
	height := header.Height()

	txns := make([]*coinharness.Tx, 0, len(filteredTxns))
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
		m.chainUpdateSignal <- chainUpdateSignal
	}()
}

//// ingestBlock updates the wallet's internal utxo state based on the outputs
//// created and destroyed within each block.
//func (wallet *InMemoryWallet) ingestBlock(update *chainUpdate) {
//	// Update the latest synced height, then process each filtered
//	// transaction in the block creating and destroying utxos within
//	// the wallet as a result.
//	wallet.currentHeight = update.blockHeight
//	undo := &undoEntry{
//		utxosDestroyed: make(map[coinharness.OutPoint]*utxo),
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
//	wallet.reorgJournal[update.blockHeight] = undo
//}

// chainSyncer is a goroutine dedicated to processing new blocks in order to
// keep the wallet's utxo state up to date.
//
// NOTE: This MUST be run as a goroutine.
func (wallet *InMemoryWallet) chainSyncer() {
	var update *chainUpdate

	for s := range wallet.chainUpdateSignal {
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
		// transaction in the block creating and destroying utxos within
		// the wallet as a result.
		wallet.Lock()
		wallet.currentHeight = update.blockHeight
		undo := &undoEntry{
			utxosDestroyed: make(map[coinharness.OutPoint]*utxo),
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
		wallet.reorgJournal[update.blockHeight] = undo
		wallet.Unlock()
	}
}

// evalOutputs evaluates each of the passed outputs, creating a new matching
// utxo within the wallet if we're able to spend the output.
func (wallet *InMemoryWallet) evalOutputs(outputs []*coinharness.TxOut, txHash coinharness.Hash, isCoinbase bool, undo *undoEntry) {
	for i, output := range outputs {
		pkScript := output.PkScript

		// Scan all the addresses we currently control to see if the
		// output is paying to us.
		for keyIndex, addr := range wallet.addrs {
			pkHash := addr.ScriptAddress()
			if !bytes.Contains(pkScript, pkHash) {
				continue
			}

			// If this is a coinbase output, then we mark the
			// maturity height at the proper block height in the
			// future.
			var maturityHeight int64
			if isCoinbase {
				maturityHeight = wallet.currentHeight + int64(wallet.net.CoinbaseMaturity())
			}

			op := coinharness.OutPoint{Hash: txHash, Index: uint32(i)}
			wallet.utxos[op] = &utxo{
				value:          output.Value.Copy(),
				keyIndex:       keyIndex,
				maturityHeight: maturityHeight,
				pkScript:       pkScript,
			}
			undo.utxosCreated = append(undo.utxosCreated, op)
		}
	}
}

// evalInputs scans all the passed inputs, destroying any utxos within the
// wallet which are spent by an input.
func (wallet *InMemoryWallet) evalInputs(inputs []*coinharness.TxIn, undo *undoEntry) {
	for _, txIn := range inputs {
		op := txIn.PreviousOutPoint
		oldUtxo, ok := wallet.utxos[op]
		if !ok {
			continue
		}

		undo.utxosDestroyed[op] = oldUtxo
		delete(wallet.utxos, op)
	}
}

// UnwindBlock is a call-back which is to be executed each time a block is
// disconnected from the main chain. Unwinding a block undoes the effect that a
// particular block had on the wallet's internal utxo state.
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

	undo := m.reorgJournal[height]

	for _, utxo := range undo.utxosCreated {
		delete(m.utxos, utxo)
	}

	for outPoint, utxo := range undo.utxosDestroyed {
		m.utxos[outPoint] = utxo
	}

	delete(m.reorgJournal, height)
}

// unwindBlock undoes the effect that a particular block had on the wallet's
// internal utxo state.
func (wallet *InMemoryWallet) unwindBlock(update *chainUpdate) {
	undo := wallet.reorgJournal[update.blockHeight]

	for _, utxo := range undo.utxosCreated {
		delete(wallet.utxos, utxo)
	}

	for outPoint, utxo := range undo.utxosDestroyed {
		wallet.utxos[outPoint] = utxo
	}

	delete(wallet.reorgJournal, update.blockHeight)
}

// newAddress returns a new address from the wallet's hd key chain.  It also
// loads the address into the RPC client's transaction filter to ensure any
// transactions that involve it are delivered via the notifications.
func (wallet *InMemoryWallet) newAddress() (coinharness.Address, error) {
	index := wallet.hdIndex

	childKey, err := wallet.hdRoot.Child(index)
	if err != nil {
		return nil, err
	}
	privKey, err := childKey.PrivateKey()
	if err != nil {
		return nil, err
	}

	addr, err := wallet.PrivateKeyKeyToAddr(privKey, wallet.net)
	if err != nil {
		return nil, err
	}

	err = wallet.nodeRPC.LoadTxFilter(false, []coinharness.Address{addr})
	if err != nil {
		return nil, err
	}

	wallet.addrs[index] = addr

	wallet.hdIndex++

	return addr, nil
}

// NewAddress returns a fresh address spendable by the wallet.
//
// This function is safe for concurrent access.
func (wallet *InMemoryWallet) NewAddress(account string) (coinharness.Address, error) {
	wallet.Lock()
	defer wallet.Unlock()

	add, err := wallet.newAddress()

	if err != nil {
		return nil, err
	}

	return add, nil
}

func (wallet *InMemoryWallet) ListUnspent() (result []*coinharness.Unspent, err error) {
	wallet.Lock()
	defer wallet.Unlock()

	panic("not implemented")
}

// UnlockOutputs unlocks any outputs which were previously locked due to
// being selected to fund a transaction via the CreateTransaction method.
//
// This function is safe for concurrent access.
func (wallet *InMemoryWallet) UnlockOutputs(inputs []coinharness.TxIn) error {
	wallet.Lock()
	defer wallet.Unlock()

	for _, input := range inputs {
		utxo, ok := wallet.utxos[input.PreviousOutPoint]
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
func (wallet *InMemoryWallet) GetBalance() (*coinharness.GetBalanceResult, error) {
	wallet.RLock()
	defer wallet.RUnlock()
	result := &coinharness.GetBalanceResult{}
	//result.BlockHash
	b := coinharness.GetAccountBalanceResult{}
	result.Balances[coinharness.DefaultAccountName] = b

	balance := coinharness.CoinsAmount{0}
	for _, utxo := range wallet.utxos {
		// Prevent any immature or locked outputs from contributing to
		// the wallet's total confirmed balance.
		if !utxo.isMature(wallet.currentHeight) || utxo.isLocked {
			continue
		}

		balance.AtomsValue += utxo.value.AtomsValue
	}

	b.Spendable = balance
	b.AccountName = coinharness.DefaultAccountName
	return result, nil
}

func (wallet *InMemoryWallet) RPCClient() *coinharness.RPCConnection {
	panic("Method not supported")
}

func (wallet *InMemoryWallet) CreateNewAccount(accountName string) error {
	panic("")
}

func (wallet *InMemoryWallet) GetNewAddress(accountName string) (coinharness.Address, error) {
	panic("")
}
func (wallet *InMemoryWallet) ValidateAddress(address coinharness.Address) (*coinharness.ValidateAddressResult, error) {
	panic("")
}

func (wallet *InMemoryWallet) WalletUnlock(password string, seconds int64) error {
	return nil
}
func (wallet *InMemoryWallet) WalletInfo() (*coinharness.WalletInfoResult, error) {
	return &coinharness.WalletInfoResult{
		Unlocked: true,
	}, nil
}
func (wallet *InMemoryWallet) WalletLock() error {
	return nil
}
