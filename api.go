package coinharness

import "fmt"

type Network interface {
	CoinbaseMaturity() int64
	Params() interface{}
}

type RPCConnectionConfig struct {
	Host            string
	Endpoint        string
	User            string
	Pass            string
	CertificateFile string
}

type Address interface {
	String() string
	IsForNet(network Network) bool
	Internal() interface{}
	ScriptAddress() []byte
}

type Seed interface{}

type TxIn struct {
	PreviousOutPoint OutPoint
	ValueIn          CoinsAmount

	// Non-witness
	Sequence uint32

	// Witness
	BlockHeight     uint32
	BlockIndex      uint32
	SignatureScript []byte
}

type OutPoint struct {
	Hash  Hash
	Index uint32
	Tree  int8
}

type Hash interface{}

type TxOut struct {
	Version  uint16
	PkScript []byte
	Value    CoinsAmount
}

type CoinsAmount struct {
	AtomsValue int64
}

func (a CoinsAmount) String() string {
	return fmt.Sprintf("%v coins", a.ToCoins())
}

func CoinsAmountFromFloat(coinsFloat float64) CoinsAmount {
	return CoinsAmount{int64(coinsFloat * 1e8)}
}

func (a *CoinsAmount) ToCoins() float64 {
	return float64(a.AtomsValue) / 1e8
}

func (a *CoinsAmount) ToAtoms() int64 {
	return a.AtomsValue
}

func (a *CoinsAmount) Copy() CoinsAmount {
	return CoinsAmount{a.AtomsValue}
}

type MessageTx struct {
	//CachedHash Hash
	SerType  uint16
	Version  int32
	TxIn     []*TxIn
	TxOut    []*TxOut
	LockTime uint32
	Expiry   uint32
	TxHash   func() Hash
}

type Tx struct {
	Hash    Hash       // Cached transaction hash
	MsgTx   *MessageTx // Underlying MsgTx
	TxTree  int8       // Indicates which tx tree the tx is found in
	TxIndex int        // Position within a block or TxIndexUnknown
}

type AddNodeArguments struct {
	TargetAddr string
	Command    interface{}
}

type CoinbaseKey interface{}

type ExtendedKey interface {
	Child(u uint32) (ExtendedKey, error)
	PrivateKey() (PrivateKey, error)
}

type BlockHeader interface {
	Height() int64
}

type PublicKey interface{}

type PrivateKey interface {
	PublicKey() PublicKey
}
