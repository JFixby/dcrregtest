package dcrregtest

import (
	"fmt"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrjson"
	"github.com/decred/dcrd/dcrutil"
	"github.com/decred/dcrd/rpcclient"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrwallet/errors"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"math"
	"testing"
	"time"
)

func mineBlock(t *testing.T, r *coinharness.Harness) {
	_, heightBefore, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlock()
	if err != nil {
		t.Fatal("Failed to get chain height:", err)
	}

	err = coinharness.GenerateTestChain(1, r.NodeRPCClient())
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
	count := r.Wallet.Sync(heightAfter)
	if heightAfter != count {
		t.Fatal("Failed to sync wallet to target:", heightAfter)
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

// GenerateBlock is a helper function to ensure that the chain has actually
// incremented due to FORK blocks after stake voting height that may occur.
func GenerateBlock(h *coinharness.Harness, startHeight uint32) ([]*chainhash.Hash, error) {
	blockHashes, err := h.NodeRPCClient().Internal().(*rpcclient.Client).Generate(1)
	if err != nil {
		return nil, errors.Errorf("unable to generate single block: %v", err)
	}
	blockHeader, err := h.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockHeader(blockHashes[0])
	if err != nil {
		return nil, errors.Errorf("unable to get block header: %v", err)
	}
	newHeight := blockHeader.Height
	for newHeight == startHeight {
		blockHashes, err = h.NodeRPCClient().Internal().(*rpcclient.Client).Generate(1)
		if err != nil {
			return nil, errors.Errorf("unable to generate single block: %v", err)
		}
		blockHeader, err = h.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockHeader(blockHashes[0])
		if err != nil {
			return nil, errors.Errorf("unable to get block header: %v", err)
		}
		newHeight = blockHeader.Height
	}
	return blockHashes, nil
}

func mustGetStakeInfo(wcl *rpcclient.Client, t *testing.T) *dcrjson.GetStakeInfoResult {
	stakeinfo, err := wcl.GetStakeInfo()
	if err != nil {
		t.Fatal("GetStakeInfo failed: ", err)
	}
	return stakeinfo
}

func mustGetStakeDiff(r *coinharness.Harness, t *testing.T) float64 {
	stakeDiffResult, err := r.WalletRPCClient().Internal().(*rpcclient.Client).GetStakeDifficulty()
	if err != nil {
		t.Fatal("GetStakeDifficulty failed:", err)
	}

	return stakeDiffResult.CurrentStakeDifficulty
}

func mustGetStakeDiffNext(r *coinharness.Harness, t *testing.T) float64 {
	stakeDiffResult, err := r.WalletRPCClient().Internal().(*rpcclient.Client).GetStakeDifficulty()
	if err != nil {
		t.Fatal("GetStakeDifficulty failed:", err)
	}

	return stakeDiffResult.NextStakeDifficulty
}

func advanceToHeight(r *coinharness.Harness, t *testing.T, height uint32) {
	curBlockHeight := getBestBlockHeight(r, t)
	initHeight := curBlockHeight

	if curBlockHeight >= height {
		return
	}

	for curBlockHeight != height {
		curBlockHeight, _, _ = newBlockAtQuick(curBlockHeight, r, t)
		time.Sleep(75 * time.Millisecond)
	}
	t.Logf("Advanced %d blocks to block height %d", curBlockHeight-initHeight,
		curBlockHeight)
}

func newBlockAt(currentHeight uint32, r *coinharness.Harness,
	t *testing.T) (uint32, *dcrutil.Block, []*chainhash.Hash) {
	height, block, blockHashes := newBlockAtQuick(currentHeight, r, t)

	time.Sleep(700 * time.Millisecond)

	return height, block, blockHashes
}

func newBlockAtQuick(currentHeight uint32, r *coinharness.Harness,
	t *testing.T) (uint32, *dcrutil.Block, []*chainhash.Hash) {

	blockHashes, err := GenerateBlock(r, currentHeight)
	if err != nil {
		t.Fatalf("Unable to generate single block: %v", err)
	}

	block, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlock(blockHashes[0])
	if err != nil {
		t.Fatalf("Unable to get block: %v", err)
	}

	return block.Header.Height, dcrutil.NewBlock(block), blockHashes
}

func getBestBlock(r *coinharness.Harness, t *testing.T) (uint32, *dcrutil.Block, *chainhash.Hash) {
	bestBlockHash, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlockHash()
	if err != nil {
		t.Fatalf("Unable to get best block hash: %v", err)
	}
	bestBlock, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlock(bestBlockHash)
	if err != nil {
		t.Fatalf("Unable to get block: %v", err)
	}
	curBlockHeight := bestBlock.Header.Height

	return curBlockHeight, dcrutil.NewBlock(bestBlock), bestBlockHash
}

func getBestBlockHeight(r *coinharness.Harness, t *testing.T) uint32 {
	_, height, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlock()
	if err != nil {
		t.Fatalf("Failed to GetBestBlock: %v", err)
	}

	return uint32(height)
}

func newBestBlock(r *coinharness.Harness,
	t *testing.T) (uint32, *dcrutil.Block, []*chainhash.Hash) {
	height := getBestBlockHeight(r, t)
	height, block, blockHash := newBlockAt(height, r, t)
	return height, block, blockHash
}

// includesTx checks if a block contains a transaction hash
func includesTx(txHash *chainhash.Hash, block *dcrutil.Block) bool {
	if len(block.Transactions()) <= 1 {
		return false
	}

	blockTxs := block.Transactions()

	for _, minedTx := range blockTxs {
		minedTxHash := minedTx.Hash()
		if *txHash == *minedTxHash {
			return true
		}
	}

	return false
}

// includesTx checks if a block contains a transaction hash
func includesStakeTx(txHash *chainhash.Hash, block *dcrutil.Block) bool {
	if len(block.STransactions()) <= 1 {
		return false
	}

	blockTxs := block.STransactions()

	for _, minedTx := range blockTxs {
		minedTxHash := minedTx.Hash()
		if *txHash == *minedTxHash {
			return true
		}
	}

	return false
}

// getWireMsgTxFee computes the effective absolute fee from a Tx as the amount
// spent minus sent.
func getWireMsgTxFee(tx *dcrutil.Tx) dcrutil.Amount {
	var totalSpent int64
	for _, txIn := range tx.MsgTx().TxIn {
		totalSpent += txIn.ValueIn
	}

	var totalSent int64
	for _, txOut := range tx.MsgTx().TxOut {
		totalSent += txOut.Value
	}

	return dcrutil.Amount(totalSpent - totalSent)
}

// getOutPointString uses OutPoint.String() to combine the tx hash with vout
// index from a ListUnspentResult.
func getOutPointString(utxo *dcrjson.ListUnspentResult) (string, error) {
	txhash, err := chainhash.NewHashFromStr(utxo.TxID)
	if err != nil {
		return "", err
	}
	return wire.NewOutPoint(txhash, utxo.Vout, utxo.Tree).String(), nil
}
