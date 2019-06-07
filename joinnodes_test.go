// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dcrregtest

import (
	"github.com/decred/dcrd/dcrutil"
	"github.com/decred/dcrd/rpcclient"
	"github.com/jfixby/cointest"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrjson"
	"github.com/decred/dcrd/txscript"
	"github.com/decred/dcrd/wire"
)

func TestJoinBlocks(t *testing.T) {
	// Skip tests when running with -short
	//if testing.Short() {
	//	t.Skip("Skipping RPC harness tests in short mode")
	//}
	r := ObtainHarness(mainHarnessName)

	// Create a second harness with only the genesis block so it is behind
	// the main harness.
	h := testSetup.Regnet0.NewInstance("checkJoinBlocks").(*cointest.Harness)
	defer testSetup.Regnet0.Dispose(h)

	nodeSlice := []*cointest.Harness{r, h}
	blocksSynced := make(chan struct{})
	go func() {
		if err := JoinNodes(nodeSlice, Blocks); err != nil {
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
	if err := ConnectNode(h, r); err != nil {
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
func TestJoinMempools(t *testing.T) {
	// Skip tests when running with -short
	//if testing.Short() {
	//	t.Skip("Skipping RPC harness tests in short mode")
	//}
	r := ObtainHarness(mainHarnessName)

	// Assert main test harness has no transactions in its mempool.
	pooledHashes, err := r.NodeRPCClient().(*rpcclient.Client).GetRawMempool(dcrjson.GRMAll)
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
	h := testSetup.Regnet0.NewInstance("checkJoinMempools").(*cointest.Harness)
	defer testSetup.Regnet0.Dispose(h)

	nodeSlice := []*cointest.Harness{r, h}

	// Both mempools should be considered synced as they are empty.
	// Therefore, this should return instantly.
	if err := JoinNodes(nodeSlice, Mempools); err != nil {
		t.Fatalf("unable to join node on mempools: %v", err)
	}

	// Generate a coinbase spend to a new address within the main harness'
	// mempool.
	addr, err := r.Wallet.NewAddress(nil)
	if err != nil {
		t.Fatalf("unable to generate address: %v", err)
	}
	addrScript, err := txscript.PayToAddrScript(addr.(dcrutil.Address))
	if err != nil {
		t.Fatalf("unable to generate pkscript to addr: %v", err)
	}

	output := wire.NewTxOut(5e8, addrScript)
	ctargs := &cointest.CreateTransactionArgs{
		Outputs: []cointest.OutputTx{output},
		FeeRate: 10,
	}
	testTx, err := r.Wallet.CreateTransaction(ctargs)
	if err != nil {
		t.Fatalf("coinbase spend failed: %v", err)
	}
	if _, err := r.NodeRPCClient().(*rpcclient.Client).SendRawTransaction(testTx.(*wire.MsgTx), true); err != nil {
		t.Fatalf("send transaction failed: %v", err)
	}

	// Wait until the transaction shows up to ensure the two mempools are
	// not the same.
	harnessSynced := make(chan struct{})
	go func() {
		for {
			poolHashes, err := r.NodeRPCClient().(*rpcclient.Client).GetRawMempool(dcrjson.GRMAll)
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
		if err := JoinNodes(nodeSlice, Mempools); err != nil {
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
	if err := ConnectNode(h, r); err != nil {
		t.Fatalf("unable to connect harnesses: %v", err)
	}
	if err := JoinNodes(nodeSlice, Blocks); err != nil {
		t.Fatalf("unable to join node on blocks: %v", err)
	}

	// Send the transaction to the local harness which will result in synced
	// mempools.
	if _, err := h.NodeRPCClient().(*rpcclient.Client).SendRawTransaction(testTx.(*wire.MsgTx), true); err != nil {
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
