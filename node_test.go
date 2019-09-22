package dcrregtest

import (
	"bytes"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrjson"
	"github.com/decred/dcrd/dcrutil"
	"github.com/decred/dcrd/rpcclient"
	"github.com/decred/dcrd/txscript"
	"github.com/decred/dcrd/wire"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/dcrharness"
	"testing"
	"time"
)

func TestGetBestBlock(t *testing.T) {
	r := ObtainHarness(mainHarnessName)

	_, prevbestHeight, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlock()
	if err != nil {
		t.Fatalf("Call to `getbestblock` failed: %v", err)
	}

	// Create a new block connecting to the current tip.
	generatedBlockHashes, err := r.NodeRPCClient().Generate(1)
	if err != nil {
		t.Fatalf("Unable to generate block: %v", err)
	}

	bestHash, bestHeight, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlock()
	if err != nil {
		t.Fatalf("Call to `getbestblock` failed: %v", err)
	}

	// Hash should be the same as the newly submitted block.
	b1 := bestHash[:]
	b2 := generatedBlockHashes[0].(*chainhash.Hash)[:]
	if !bytes.Equal(b1, b2) {
		t.Fatalf("Block hashes do not match. Returned hash %v, wanted "+
			"hash %v", b1, b2)
	}

	// Block height should now reflect newest height.
	if bestHeight != prevbestHeight+1 {
		t.Fatalf("Block heights do not match. Got %v, wanted %v",
			bestHeight, prevbestHeight+1)
	}
}

func TestGetBlockCount(t *testing.T) {
	r := ObtainHarness(mainHarnessName)
	// Save the current count.
	currentCount, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockCount()
	if err != nil {
		t.Fatalf("Unable to get block count: %v", err)
	}

	if _, err := r.NodeRPCClient().Generate(1); err != nil {
		t.Fatalf("Unable to generate block: %v", err)
	}

	// Count should have increased by one.
	newCount, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockCount()
	if err != nil {
		t.Fatalf("Unable to get block count: %v", err)
	}
	if newCount != currentCount+1 {
		t.Fatalf("Block count incorrect. Got %v should be %v",
			newCount, currentCount+1)
	}
}

func TestGetBlockHash(t *testing.T) {
	r := ObtainHarness(mainHarnessName)
	// Create a new block connecting to the current tip.
	generatedBlockHashes, err := r.NodeRPCClient().Generate(1)
	if err != nil {
		t.Fatalf("Unable to generate block: %v", err)
	}

	info, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetInfo()
	if err != nil {
		t.Fatalf("call to getinfo cailed: %v", err)
	}

	blockHash, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockHash(int64(info.Blocks))
	if err != nil {
		t.Fatalf("Call to `getblockhash` failed: %v", err)
	}

	// Block hashes should match newly created block.
	if !bytes.Equal(generatedBlockHashes[0].(*chainhash.Hash)[:], blockHash[:]) {
		t.Fatalf("Block hashes do not match. Returned hash %v, wanted "+
			"hash %v", blockHash, generatedBlockHashes[0].(*chainhash.Hash)[:])
	}
}

func TestConnectNode(t *testing.T) {
	r := ObtainHarness(mainHarnessName)

	// Create a fresh test harness.
	harness := testSetup.Regnet0.NewInstance(t.Name()).(*coinharness.Harness)
	defer testSetup.Regnet0.Dispose(harness)

	// Establish a p2p connection from our new local harness to the main
	// harness.
	if err := coinharness.ConnectNode(harness, r, rpcclient.ANAdd); err != nil {
		t.Fatalf("unable to connect local to main harness: %v", err)
	}

	// The main harness should show up in our local harness' peer's list,
	// and vice verse.
	coinharness.AssertConnectedTo(t, harness, r)
}

func TestJoin(t *testing.T) {
	checkJoinBlocks(t)
	checkJoinMempools(t)
}

func checkJoinBlocks(t *testing.T) {
	r := ObtainHarness(mainHarnessName)

	// Create a second harness with only the genesis block so it is behind
	// the main harness.
	h := testSetup.Regnet0.NewInstance("checkJoinBlocks").(*coinharness.Harness)
	defer testSetup.Regnet0.Dispose(h)

	nodeSlice := []*coinharness.Harness{r, h}
	blocksSynced := make(chan struct{})
	go func() {
		if err := coinharness.JoinNodes(dcrjson.GRMAll, nodeSlice, coinharness.Blocks); err != nil {
			t.Fatalf("unable to join node on blocks: %v", err)
		}
		blocksSynced <- struct{}{}
	}()

	// This select case should fall through to the default as the goroutine
	// should be blocked on the JoinNodes calls.
	select {
	case <-blocksSynced:
		t.Fatalf("blocks detected as synced yet local harness is behind")
	default:
	}

	// Connect the local harness to the main harness which will sync the
	// chains.
	if err := coinharness.ConnectNode(h, r, rpcclient.ANAdd); err != nil {
		t.Fatalf("unable to connect harnesses: %v", err)
	}

	// Select once again with a special timeout case after 1 minute. The
	// goroutine above should now be blocked on sending into the unbuffered
	// channel. The send should immediately succeed. In order to avoid the
	// test hanging indefinitely, a 1 minute timeout is in place.
	select {
	case <-blocksSynced:
		// fall through
	case <-time.After(time.Minute):
		t.Fatalf("blocks never detected as synced")
	}
}

// TestJoinMempools must be executed after the TestJoinBlocks
func checkJoinMempools(t *testing.T) {
	r := ObtainHarness(mainHarnessName)

	// Assert main test harness has no transactions in its mempool.
	pooledHashes, err := r.NodeRPCClient().GetRawMempool(dcrjson.GRMAll)
	if err != nil {
		t.Fatalf("unable to get mempool for main test harness: %v", err)
	}
	if len(pooledHashes) != 0 {
		t.Fatal("main test harness mempool not empty")
	}

	// Create a local test harness with only the genesis block.  The nodes
	// will be synced below so the same transaction can be sent to both
	// nodes without it being an orphan.
	// Create a fresh test harness.
	h := testSetup.Regnet0.NewInstance("checkJoinMempools").(*coinharness.Harness)
	defer testSetup.Regnet0.Dispose(h)

	nodeSlice := []*coinharness.Harness{r, h}

	// Both mempools should be considered synced as they are empty.
	// Therefore, this should return instantly.
	if err := coinharness.JoinNodes(dcrjson.GRMAll, nodeSlice, coinharness.Mempools); err != nil {
		t.Fatalf("unable to join node on mempools: %v", err)
	}

	// Generate a coinbase spend to a new address within the main harness'
	// mempool.
	addr, err := r.Wallet.NewAddress(nil)
	if err != nil {
		t.Fatalf("unable to generate address: %v", err)
	}
	addrScript, err := txscript.PayToAddrScript(addr.Internal().(dcrutil.Address))
	if err != nil {
		t.Fatalf("unable to generate pkscript to addr: %v", err)
	}

	output := &coinharness.TxOut{
		Amount:   coinharness.CoinsAmountFromFloat(5),
		PkScript: addrScript,
		Version:  wire.DefaultPkScriptVersion,
	}
	ctargs := &coinharness.CreateTransactionArgs{
		Outputs:         []*coinharness.TxOut{output},
		FeeRate:         coinharness.CoinsAmount{10},
		PayToAddrScript: dcrharness.PayToAddrScript,
		TxSerializeSize: dcrharness.TxSerializeSize,
	}
	testTx, err := coinharness.CreateTransaction(r.Wallet, ctargs)
	if err != nil {
		t.Fatalf("coinbase spend failed: %v", err)
	}
	if _, err := r.NodeRPCClient().SendRawTransaction(testTx, true); err != nil {
		t.Fatalf("send transaction failed: %v", err)
	}

	// Wait until the transaction shows up to ensure the two mempools are
	// not the same.
	harnessSynced := make(chan struct{})
	go func() {
		for {
			poolHashes, err := r.NodeRPCClient().GetRawMempool(dcrjson.GRMAll)
			if err != nil {
				t.Fatalf("failed to retrieve harness mempool: %v", err)
			}
			if len(poolHashes) > 0 {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
		harnessSynced <- struct{}{}
	}()
	select {
	case <-harnessSynced:
	case <-time.After(time.Minute):
		t.Fatalf("harness node never received transaction")
	}

	// This select case should fall through to the default as the goroutine
	// should be blocked on the JoinNodes call.
	poolsSynced := make(chan struct{})
	go func() {
		if err := coinharness.JoinNodes(dcrjson.GRMAll, nodeSlice, coinharness.Mempools); err != nil {
			t.Fatalf("unable to join node on mempools: %v", err)
		}
		poolsSynced <- struct{}{}
	}()
	select {
	case <-poolsSynced:
		t.Fatalf("mempools detected as synced yet harness has a new tx")
	default:
	}

	// Establish an outbound connection from the local harness to the main
	// harness and wait for the chains to be synced.
	if err := coinharness.ConnectNode(h, r, rpcclient.ANAdd); err != nil {
		t.Fatalf("unable to connect harnesses: %v", err)
	}
	if err := coinharness.JoinNodes(dcrjson.GRMAll, nodeSlice, coinharness.Blocks); err != nil {
		t.Fatalf("unable to join node on blocks: %v", err)
	}

	// Send the transaction to the local harness which will result in synced
	// mempools.
	if _, err := h.NodeRPCClient().SendRawTransaction(testTx, true); err != nil {
		t.Fatalf("send transaction failed: %v", err)
	}

	// Select once again with a special timeout case after 1 minute. The
	// goroutine above should now be blocked on sending into the unbuffered
	// channel. The send should immediately succeed. In order to avoid the
	// test hanging indefinitely, a 1 minute timeout is in place.
	select {
	case <-poolsSynced:
		// fall through
	case <-time.After(time.Minute):
		t.Fatalf("mempools never detected as synced")
	}
}

func TestMemWalletLockedOutputs(t *testing.T) {
	r := ObtainHarness(mainHarnessName)
	// Obtain the initial balance of the wallet at this point.
	startingBalance := coinharness.GetBalance(t, r.Wallet).TotalSpendable

	// First, create a signed transaction spending some outputs.
	addr, err := r.Wallet.NewAddress(nil)
	if err != nil {
		t.Fatalf("unable to generate new address: %v", err)
	}
	pkScript, err := txscript.PayToAddrScript(addr.Internal().(dcrutil.Address))
	if err != nil {
		t.Fatalf("unable to create script: %v", err)
	}

	outputAmt := coinharness.CoinsAmountFromFloat(50)
	output := &coinharness.TxOut{
		Amount:   outputAmt,
		PkScript: pkScript,
		Version:  wire.DefaultPkScriptVersion,
	}
	ctargs := &coinharness.CreateTransactionArgs{
		Outputs:         []*coinharness.TxOut{output},
		FeeRate:         coinharness.CoinsAmount{10},
		PayToAddrScript: dcrharness.PayToAddrScript,
		TxSerializeSize: dcrharness.TxSerializeSize,
	}
	_, err = coinharness.CreateTransaction(r.Wallet, ctargs)
	if err != nil {
		t.Fatalf("unable to create transaction: %v", err)
	}

	// The current wallet balance should now be at least 50 BTC less
	// (accounting for fees) than the period balance
	currentBalance := coinharness.GetBalance(t, r.Wallet).TotalSpendable
	if !(currentBalance.AtomsValue <= startingBalance.AtomsValue-outputAmt.AtomsValue) {
		t.Fatalf("spent outputs not locked: previous balance %v, "+
			"current balance %v", startingBalance, currentBalance)
	}

	// Now unlocked all the spent inputs within the unbroadcast signed
	// transaction. The current balance should now be exactly that of the
	// starting balance.
	//txin := tx.TxIn
	//r.Wallet.UnlockOutputs(txin)
	currentBalance = coinharness.GetBalance(t, r.Wallet).TotalSpendable
	if currentBalance != startingBalance {
		t.Fatalf("current and starting balance should now match: "+
			"expected %v, got %v", startingBalance, currentBalance)
	}
}

func TestP2PConnect(t *testing.T) {
	r := ObtainHarness(mainHarnessName)

	// Create a fresh test harness.
	harness := testSetup.Regnet25.NewInstance("TestP2PConnect").(*coinharness.Harness)
	defer testSetup.Regnet25.Dispose(harness)

	// Establish a p2p connection from our new local harness to the main
	// harness.
	if err := coinharness.ConnectNode(harness, r, rpcclient.ANAdd); err != nil {
		t.Fatalf("unable to connect local to main harness: %v", err)
	}

	// The main harness should show up in our local harness' peer's list,
	// and vice verse.
	coinharness.AssertConnectedTo(t, harness, r)
}

func TestMemWalletReorg(t *testing.T) {
	r := ObtainHarness(mainHarnessName)

	// Create a fresh h, we'll be using the main h to force a
	// re-org on this local h.
	h := testSetup.Regnet5.NewInstance(t.Name() + ".4").(*coinharness.Harness)
	defer testSetup.Regnet5.Dispose(h)
	h.Wallet.Sync(testSetup.Regnet5.NumMatureOutputs)

	expectedBalance := coinharness.CoinsAmountFromFloat(1200)
	walletBalance := coinharness.GetBalance(t, h.Wallet).TotalSpendable
	if expectedBalance.AtomsValue != walletBalance.AtomsValue {
		t.Fatalf("wallet balance incorrect: expected %v, got %v",
			expectedBalance, walletBalance)
	}

	// Now connect this local h to the main h then wait for
	// their chains to synchronize.
	if err := coinharness.ConnectNode(h, r, rpcclient.ANAdd); err != nil {
		t.Fatalf("unable to connect harnesses: %v", err)
	}
	nodeSlice := []*coinharness.Harness{r, h}
	if err := coinharness.JoinNodes(dcrjson.GRMAll, nodeSlice, coinharness.Blocks); err != nil {
		t.Fatalf("unable to join node on blocks: %v", err)
	}

	// The original wallet should now have a balance of 0 BTC as its entire
	// chain should have been decimated in favor of the main h'
	// chain.
	expectedBalance = coinharness.CoinsAmountFromFloat(0)
	walletBalance = coinharness.GetBalance(t, h.Wallet).TotalSpendable
	if expectedBalance.AtomsValue != walletBalance.AtomsValue {
		t.Fatalf("wallet balance incorrect: expected %v, got %v",
			expectedBalance, walletBalance)
	}
}
