package coinharness

import (
	"fmt"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
	"strconv"
	"strings"
)

// DeploySimpleChain defines harness setup sequence for this package:
// 1. obtains a new mining wallet address
// 2. restart harness node and wallet with the new mining address
// 3. builds a new chain with the target number of mature outputs
// receiving the mining reward to the test wallet
// 4. syncs wallet to the tip of the chain
func DeploySimpleChain(testSetup *ChainWithMatureOutputsSpawner, h *Harness) {
	pin.AssertNotEmpty("harness name", h.Name)
	fmt.Println("Deploying Harness[" + h.Name + "]")
	createFlag := testSetup.CreateTempWallet
	// launch a fresh h (assumes h working dir is empty)
	{
		args := &launchArguments{
			DebugNodeOutput:    testSetup.DebugNodeOutput,
			DebugWalletOutput:  testSetup.DebugWalletOutput,
			NodeExtraArguments: testSetup.NodeStartExtraArguments,
		}
		if createFlag {
			args.WalletExtraArguments = make(map[string]interface{})
			args.WalletExtraArguments["createtemp"] = commandline.NoArgumentValue
		}
		launchHarnessSequence(h, args)
	}

	// Get a new address from the WalletTestServer
	// to be set with node --miningaddr
	var address Address
	var err error
	{
		for {
			address, err = h.Wallet.NewAddress(DefaultAccountName)
			if err != nil {
				pin.D("address", address)
				pin.D("error", err)
				pin.Sleep(1000)
			} else {
				break
			}
		}

		//pin.CheckTestSetupMalfunction(err)
		h.MiningAddress = address

		pin.AssertNotNil("MiningAddress", h.MiningAddress)
		pin.AssertNotEmpty("MiningAddress", h.MiningAddress.String())

		fmt.Println("Mining address: " + h.MiningAddress.String())
	}

	// restart the h with the new argument
	{
		shutdownHarnessSequence(h)

		args := &launchArguments{
			DebugNodeOutput:    testSetup.DebugNodeOutput,
			DebugWalletOutput:  testSetup.DebugWalletOutput,
			NodeExtraArguments: testSetup.NodeStartExtraArguments,
		}
		if createFlag {
			args.WalletExtraArguments = make(map[string]interface{})
			args.WalletExtraArguments["createtemp"] = commandline.NoArgumentValue
		}
		launchHarnessSequence(h, args)
	}

	{
		if testSetup.NumMatureOutputs > 0 {
			numToGenerate := int64(testSetup.ActiveNet.CoinbaseMaturity()) + testSetup.NumMatureOutputs
			err := GenerateTestChain(numToGenerate, h.NodeRPCClient())
			pin.CheckTestSetupMalfunction(err)
		}
		// wait for the WalletTestServer to sync up to the current height
		_, H, e := h.NodeRPCClient().GetBestBlock()
		pin.CheckTestSetupMalfunction(e)
		h.Wallet.Sync(H)
	}
	fmt.Println("Harness[" + h.Name + "] is ready")
}

// local struct to bundle launchHarnessSequence function arguments
type launchArguments struct {
	DebugNodeOutput      bool
	DebugWalletOutput    bool
	MiningAddress        Address
	NodeExtraArguments   map[string]interface{}
	WalletExtraArguments map[string]interface{}
}

// launchHarnessSequence
func launchHarnessSequence(h *Harness, args *launchArguments) {
	node := h.Node
	wallet := h.Wallet

	sargs := &StartNodeArgs{
		DebugOutput:    args.DebugNodeOutput,
		MiningAddress:  h.MiningAddress,
		ExtraArguments: args.NodeExtraArguments,
	}
	node.Start(sargs)

	rpcConfig := node.RPCConnectionConfig()

	walletLaunchArguments := &TestWalletStartArgs{
		NodeRPCCertFile:          node.CertFile(),
		DebugOutput:              args.DebugWalletOutput,
		MaxSecondsToWaitOnLaunch: 90,
		NodeRPCConfig:            rpcConfig,
		ExtraArguments:           args.WalletExtraArguments,
	}

	// wait for the WalletTestServer to sync up to the current height
	_, _, e := h.NodeRPCClient().GetBestBlock()
	pin.CheckTestSetupMalfunction(e)

	wallet.Start(walletLaunchArguments)

}

// shutdownHarnessSequence reverses the launchHarnessSequence
func shutdownHarnessSequence(harness *Harness) {
	harness.Wallet.Stop()
	harness.Node.Stop()
}

// extractSeedSaltFromHarnessName tries to split harness name string
// at `.`-character and parse the second part as a uint32 number.
// Otherwise returns default value.
func extractSeedSaltFromHarnessName(harnessName string) uint32 {
	parts := strings.Split(harnessName, ".")
	if len(parts) != 2 {
		// no salt specified, return default value
		return 0
	}
	seedString := parts[1]
	tmp, err := strconv.Atoi(seedString)
	seedNonce := uint32(tmp)
	pin.CheckTestSetupMalfunction(err)
	return seedNonce
}
