package dcrregtest

import (
	"fmt"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"math"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrjson"
	"github.com/decred/dcrd/dcrutil"
	"github.com/decred/dcrd/rpcclient"
	"github.com/decred/dcrwallet/errors"
)

func mineBlock(t *testing.T, r *coinharness.Harness) {
	_, heightBefore, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlock()
	if err != nil {
		t.Fatal("Failed to get chain height:", err)
	}

	err = generateTestChain(1, r.NodeRPCClient().Internal().(*rpcclient.Client))
	if err != nil {
		t.Fatal("Failed to mine block:", err)
	}

	_, heightAfter, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlock()

	if heightAfter != heightBefore+1 {
		t.Fatal("Failed to mine block:", heightAfter, heightBefore)
	}

	if err != nil {
		t.Fatal("Failed to GetBestBlock:", err)
	}

	count, err := syncWalletTo(r.WalletRPCClient().Internal().(*rpcclient.Client), heightAfter)
	if err != nil {
		t.Fatal("Failed to sync wallet to target:", err)
	}

	if heightAfter != count {
		t.Fatal("Failed to sync wallet to target:", count)
	}
}

func reverse(results []dcrjson.ListTransactionsResult) []dcrjson.ListTransactionsResult {
	i := 0
	j := len(results) - 1
	for i < j {
		results[i], results[j] = results[j], results[i]
		i++
		j--
	}
	return results
}

//// Create a test chain with the desired number of mature coinbase outputs
//func generateTestChain(numToGenerate uint32, node *rpcclient.Client) error {
//	fmt.Printf("Generating %v blocks...\n", numToGenerate)
//	_, err := node.Generate(numToGenerate)
//	if err != nil {
//		return err
//	}
//	fmt.Println("Block generation complete.")
//	return nil
//}
//
//// Waits for wallet to sync to the target height
//func syncWalletTo(rpcClient *rpcclient.Client, desiredHeight int64) (int64, error) {
//	var count int64 = 0
//	var err error = nil
//	for count != desiredHeight {
//		Sleep(1000)
//		count, err = rpcClient.GetBlockCount()
//		if err != nil {
//			return -1, err
//		}
//		fmt.Println("   sync to: " + strconv.FormatInt(count, 10))
//	}
//	return count, nil
//}

// generateListeningPorts returns 3 subsequent network ports starting from base
func generateListeningPorts(index, base int) (int, int, int) {
	x := base + index*3 + 0
	y := base + index*3 + 1
	z := base + index*3 + 2
	return x, y, z
}

func getMiningAddr(walletClient *rpcclient.Client) dcrutil.Address {
	var miningAddr dcrutil.Address
	var err error = nil
	for i := 0; i < 100; i++ {
		miningAddr, err = walletClient.GetNewAddress("default")
		if err != nil {
			fmt.Println("err: " + err.Error())
			time.Sleep(time.Duration(math.Log(float64(i+3))) * 50 * time.Millisecond)
			continue
		}
		break
	}
	if miningAddr == nil {
		pin.ReportTestSetupMalfunction(errors.Errorf(
			"RPC not up for mining addr"))
	}
	return miningAddr
}

//// GenerateBlock is a helper function to ensure that the chain has actually
//// incremented due to FORK blocks after stake voting height that may occur.
//func (h *coinharness.Harness) GenerateBlock(startHeight uint32) ([]*chainhash.Hash, error) {
//	blockHashes, err := h.NodeRPCClient().Internal().(*rpcclient.Client).Generate(1)
//	if err != nil {
//		return nil, errors.Errorf("unable to generate single block: %v", err)
//	}
//	blockHeader, err := h.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockHeader(blockHashes[0])
//	if err != nil {
//		return nil, errors.Errorf("unable to get block header: %v", err)
//	}
//	newHeight := blockHeader.Height
//	for newHeight == startHeight {
//		blockHashes, err = h.NodeRPCClient().Internal().(*rpcclient.Client).Generate(1)
//		if err != nil {
//			return nil, errors.Errorf("unable to generate single block: %v", err)
//		}
//		blockHeader, err = h.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockHeader(blockHashes[0])
//		if err != nil {
//			return nil, errors.Errorf("unable to get block header: %v", err)
//		}
//		newHeight = blockHeader.Height
//	}
//	return blockHashes, nil
//}
