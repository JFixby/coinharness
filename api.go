package coinharness

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
}

type Seed interface{}

type InputTx interface {
	PreviousOutPoint() OutPoint
}

type OutPoint interface{}

type Hash interface{}

type OutputTx interface {
	PkScript() []byte
	Value() int64
}

type CoinsAmount int64

type CreatedTransactionTx interface {
	Version() int32
	TxIn() []InputTx
	TxOut() []OutputTx
	LockTime() uint32
	TxHash() Hash
}

type AddNodeArguments struct {
	TargetAddr string
	Command    interface{}
}
