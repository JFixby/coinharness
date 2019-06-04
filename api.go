package cointest

type Network interface{}

type RPCConnectionConfig interface{}

type ActiveNet interface{}

type Address interface{
	String() string
}

type Seed interface{}

type InputTx interface{}

type OutputTx interface{}

type CoinsAmount interface{}

type CreatedTransactionTx interface{}

type SentOutputsHash interface{}
