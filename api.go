package cointest

type Network interface{}

// RPCConnectionConfig describes the connection configuration parameters for the client.

type RPCConnectionConfig interface {
}

type RPCConnectionConfigSpawner interface {
	Spawn() RPCConnectionConfig
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
