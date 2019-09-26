package memwallet

import (
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
)

// WalletFactory produces a new InMemoryWallet-instance upon request
type WalletFactory struct {
}

// NewWallet creates and returns a fully initialized instance of the InMemoryWallet.
func (f *WalletFactory) NewWallet(cfg *coinharness.TestWalletConfig) coinharness.Wallet {
	pin.AssertNotNil("ActiveNet", cfg.ActiveNet)
	//w, e := newMemWallet(, cfg.Seed)

	net := cfg.ActiveNet
	harnessHDSeed := cfg.Seed
	PrivateKeyKeyToAddr := cfg.PrivateKeyKeyToAddr

	//hdRoot, err := hdkeychain.NewMaster(harnessHDSeed.([]byte)[:], net.Params().(*chaincfg.Params))
	hdRoot, err := cfg.NewMasterKeyFromSeed(harnessHDSeed, net)
	pin.CheckTestSetupMalfunction(err)

	// The first child key from the hd root is reserved as the coinbase
	// generation address.
	coinbaseChild, err := hdRoot.Child(0)
	pin.CheckTestSetupMalfunction(err)
	coinbaseKey, err := coinbaseChild.PrivateKey()
	pin.CheckTestSetupMalfunction(err)
	coinbaseAddr, err := PrivateKeyKeyToAddr(coinbaseKey, net)
	pin.CheckTestSetupMalfunction(err)

	// Track the coinbase generation address to ensure we properly track
	// newly generated coins we can spend.
	addrs := make(map[uint32]coinharness.Address)
	addrs[0] = coinbaseAddr

	//clientFac := &dcrharness.RPCClientFactory{}
	clientFac := cfg.RPCClientFactory
	return &InMemoryWallet{
		net:                 net,
		coinbaseKey:         coinbaseKey,
		coinbaseAddr:        coinbaseAddr,
		hdIndex:             1,
		hdRoot:              hdRoot,
		addrs:               addrs,
		utxos:               make(map[coinharness.OutPoint]*utxo),
		chainUpdateSignal:   make(chan string),
		reorgJournal:        make(map[int64]*undoEntry),
		RPCClientFactory:    clientFac,
		PrivateKeyKeyToAddr: PrivateKeyKeyToAddr,
	}

}
