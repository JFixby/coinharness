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

type InputTx struct {
	PreviousOutPoint OutPoint
	Amount           CoinsAmount
}

type OutPoint struct {
	Hash  Hash
	Index uint32
	Tree  int8
}

type Hash interface{}

type OutputTx struct {
	PkScript []byte
	Amount   CoinsAmount
}

type CoinsAmount struct {
	AtomsValue int64
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

type CreatedTransactionTx struct {
	Version  int32
	TxIn     []*InputTx
	TxOut    []*OutputTx
	LockTime uint32
	TxHash   Hash
}

type AddNodeArguments struct {
	TargetAddr string
	Command    interface{}
}
