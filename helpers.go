package coinharness

import (
	"fmt"
	"github.com/jfixby/pin"
	"reflect"
	"testing"
	"time"
)

// JoinType is an enum representing a particular type of "node join". A node
// join is a synchronization tool used to wait until a subset of nodes have a
// consistent state with respect to an attribute.
type JoinType uint8

const (
	// Blocks is a JoinType which waits until all nodes share the same
	// block height.
	Blocks JoinType = iota

	// Mempools is a JoinType which blocks until all nodes have identical
	// mempool.
	Mempools
)

// JoinNodes is a synchronization tool used to block until all passed nodes are
// fully synced with respect to an attribute. This function will block for a
// period of time, finally returning once all nodes are synced according to the
// passed JoinType. This function be used to to ensure all active test
// harnesses are at a consistent state before proceeding to an assertion or
// check within rpc tests.
func JoinNodes(command interface{}, nodes []*Harness, joinType JoinType) error {
	switch joinType {
	case Blocks:
		return syncBlocks(nodes)
	case Mempools:
		return syncMempools(command, nodes)
	}
	return nil
}

// syncMempools blocks until all nodes have identical mempools.
func syncMempools(command interface{}, nodes []*Harness) error {
	poolsMatch := false

	for !poolsMatch {
	retry:
		firstPool, err := nodes[0].NodeRPCClient().GetRawMempool(command)
		if err != nil {
			return err
		}

		// If all nodes have an identical mempool with respect to the
		// first node, then we're done. Otherwise, drop back to the top
		// of the loop and retry after a short wait period.
		for _, node := range nodes[1:] {
			nodePool, err := node.NodeRPCClient().GetRawMempool(command)
			if err != nil {
				return err
			}
			eq := reflect.DeepEqual(firstPool, nodePool)
			//eq := firstPool.EqualsTo(nodePool)
			if !eq {
				time.Sleep(time.Millisecond * 100)
				goto retry
			}
		}

		poolsMatch = true
	}

	return nil
}

// syncBlocks blocks until all nodes report the same block height.
func syncBlocks(nodes []*Harness) error {
	blocksMatch := false

	for !blocksMatch {
	retry:
		blockHeights := make(map[int64]struct{})

		for _, node := range nodes {
			blockHeight, err := node.NodeRPCClient().GetBlockCount()
			if err != nil {
				return err
			}

			blockHeights[blockHeight] = struct{}{}
			if len(blockHeights) > 1 {
				time.Sleep(time.Millisecond * 100)
				goto retry
			}
		}

		blocksMatch = true
	}

	return nil
}

// ConnectNode establishes a new peer-to-peer connection between the "from"
// harness and the "to" harness.  The connection made is flagged as persistent,
// therefore in the case of disconnects, "from" will attempt to reestablish a
// connection to the "to" harness.
func ConnectNode(from *Harness, to *Harness, command interface{}) error {
	peerInfo, err := from.NodeRPCClient().GetPeerInfo()
	if err != nil {
		return err
	}
	numPeers := len(peerInfo)

	targetAddr := to.P2PAddress()
	args := &AddNodeArguments{
		TargetAddr: targetAddr,
		//rpcclient.ANAdd,
		Command: command,
	}
	if err := from.NodeRPCClient().AddNode(args); err != nil {
		return err
	}

	// Block until a new connection has been established.
	for attempts := 5; attempts > 0; attempts-- {
		peerInfo, err = from.NodeRPCClient().GetPeerInfo()
		if err != nil {
			return err
		}
		if len(peerInfo) > numPeers {
			return nil
		}
		pin.Sleep(1000)
	}

	return fmt.Errorf("failed to connet node")
}

func AssertConnectedTo(t *testing.T, nodeA *Harness, nodeB *Harness) {
	nodeAPeers, err := nodeA.NodeRPCClient().GetPeerInfo()
	if err != nil {
		t.Fatalf("unable to get nodeA's peer info")
	}

	nodeAddr := nodeB.P2PAddress()
	addrFound := false
	for _, peerInfo := range nodeAPeers {
		if peerInfo.Addr == nodeAddr {
			addrFound = true
			break
		}
	}

	if !addrFound {
		t.Fatal("nodeA not connected to nodeB")
	}
}

// GenerateTestChain with the desired number of mature coinbase outputs
func GenerateTestChain(numToGenerate int64, node RPCClient) error {
	fmt.Printf("Generating %v blocks...\n", numToGenerate)
	_, err := node.Generate(uint32(numToGenerate))
	if err != nil {
		return err
	}
	fmt.Println("Block generation complete.")
	return nil
}

func GenSpend(
	t *testing.T,
	r *Harness,
	amt CoinsAmount,
	PkScriptVersion uint16,
	PayToAddrScript func(Address) ([]byte, error),
	TxSerializeSize func(*MessageTx) int,
) Hash {
	// Grab a fresh address from the wallet.
	addr, err := r.Wallet.NewAddress(&NewAddressArgs{"default"})
	if err != nil {
		t.Fatalf("unable to get new address: %v", err)
	}

	// Next, send amt to this address, spending from one of our
	// mature coinbase outputs.
	addrScript, err := PayToAddrScript(addr)
	if err != nil {
		t.Fatalf("unable to generate pkscript to addr: %v", err)
	}

	output := &TxOut{
		Amount:   amt,
		PkScript: addrScript,
		Version:  PkScriptVersion, //wire.DefaultPkScriptVersion
	}
	arg := &CreateTransactionArgs{
		Outputs:         []*TxOut{output},
		FeeRate:         CoinsAmountFromFloat(10),
		PayToAddrScript: PayToAddrScript,
		TxSerializeSize: TxSerializeSize,
	}

	txid, err := CreateTransaction(r.Wallet, arg)
	if err != nil {
		t.Fatalf("coinbase spend failed: %v", err)
	}
	return txid.TxHash
}

func AssertTxMined(t *testing.T, r *Harness, txid Hash, blockHash Hash) {
	block, err := r.NodeRPCClient().GetBlock(blockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}

	numBlockTxns := len(block.Transactions)
	if numBlockTxns < 2 {
		t.Fatalf("crafted transaction wasn't mined, block should have "+
			"at least %v transactions instead has %v", 2, numBlockTxns)
	}

	txHash1 := block.Transactions[1].TxHash

	if txHash1 != txid {
		t.Fatalf("txid's don't match, %v vs %v", txHash1, txid)
	}
}
