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

type OutputTx interface{}

type CoinsAmount interface{}

type CreatedTransactionTx interface{}

type SentOutputsHash interface{}
