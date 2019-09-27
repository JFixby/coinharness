package coinharness

import (
	"github.com/jfixby/coinamount"
)

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
	ValueIn          coinamount.CoinsAmount

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
	Value    coinamount.CoinsAmount
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
	Hash   Hash       // Cached transaction hash
	MsgTx  *MessageTx // Underlying MsgTx
	TxTree int8       // Indicates which tx tree the tx is found in
	Index  int        // Position within a block or TxIndexUnknown
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

type PublicKey interface {
}

type PrivateKey interface {
	PublicKey() PublicKey
}
