package dcrregtest

import (
	"github.com/decred/dcrd/wire"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/dcrharness"
	"github.com/jfixby/pin"
	"testing"
)

func TestBallance(t *testing.T) {
	// Skip tests when running with -short
	//if testing.Short() {
	//	t.Skip("Skipping RPC harness tests in short mode")
	//}
	r := ObtainHarness(t.Name() + ".8")

	expectedBalance := coinharness.CoinsAmountFromFloat(7200)
	actualBalance := coinharness.GetBalance(t, r.Wallet).TotalSpendable

	if actualBalance.AtomsValue != expectedBalance.AtomsValue {
		t.Fatalf("expected wallet balance of %v instead have %v",
			expectedBalance, actualBalance)
	}
}

func TestSendOutputs(t *testing.T) {
	// Skip tests when running with -short
	//if testing.Short() {
	//	t.Skip("Skipping RPC harness tests in short mode")
	//}
	r := ObtainHarness("TestSendOutputs")
	_, H, e := r.NodeRPCClient().GetBestBlock()
	pin.CheckTestSetupMalfunction(e)
	r.Wallet.Sync(H)
	// First, generate a small spend which will require only a single
	// input.
	txid := coinharness.GenSpend(t, r,
		coinharness.CoinsAmountFromFloat(5),
		wire.DefaultPkScriptVersion,
		dcrharness.PayToAddrScript,
		dcrharness.TxSerializeSize,
	)

	// Generate a single block, the transaction the wallet created should
	// be found in this block.
	blockHashes, err := r.NodeRPCClient().Generate(1)
	if err != nil {
		t.Fatalf("unable to generate single block: %v", err)
	}
	coinharness.AssertTxMined(t, r, txid, blockHashes[0])

	// Next, generate a spend much greater than the block reward. This
	// transaction should also have been mined properly.
	txid = coinharness.GenSpend(t, r,
		coinharness.CoinsAmountFromFloat(1000),
		wire.DefaultPkScriptVersion,
		dcrharness.PayToAddrScript,
		dcrharness.TxSerializeSize,
	)
	blockHashes, err = r.NodeRPCClient().Generate(1)
	if err != nil {
		t.Fatalf("unable to generate single block: %v", err)
	}
	coinharness.AssertTxMined(t, r, txid, blockHashes[0])

	// Generate another block to ensure the transaction is removed from the
	// mempool.
	if _, err := r.NodeRPCClient().Generate(1); err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}
}
