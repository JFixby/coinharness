package coinharness

type Network interface{}

type RPCConnectionConfig struct {
	Host            string
	Endpoint        string
	User            string
	Pass            string
	CertificateFile string
}

type ActiveNet interface{}

type Address interface {
	String() string
}

type Seed interface{}

type InputTx interface{}

type Hash interface{}

type OutputTx interface {
	PkScript() []byte
	//TxHash() Hash
	Value() int64
}

type CoinsAmount interface{}

type CreatedTransactionTx struct {
	Version  int32
	TxIn     []InputTx
	TxOut    []OutputTx
	LockTime uint32
}
type SentOutputsHash interface{}
