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

type TxIn struct {
	PreviousOutPoint OutPoint
	Amount           CoinsAmount
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

type MessageTx struct {
	Version  int32
	TxIn     []*TxIn
	TxOut    []*TxOut
	LockTime uint32
	TxHash   Hash
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
