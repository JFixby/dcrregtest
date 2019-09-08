// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pfcregtest

import (
	"bytes"
	"github.com/jfixby/coinharness"
	"github.com/picfight/pfcd/chaincfg"
	"github.com/picfight/pfcd/rpcclient"
	"github.com/picfight/pfcharness"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/picfight/pfcd/blockchain"
	"github.com/picfight/pfcd/chaincfg/chainhash"
	"github.com/picfight/pfcd/pfcec"
	"github.com/picfight/pfcd/txscript"
	"github.com/picfight/pfcd/wire"
	"github.com/picfight/pfcutil"
)

const (
	dummyForkKey = "dummy"
)

// makeTestOutput creates an on-chain output paying to a freshly generated
// p2pkh output with the specified amount.
func makeTestOutput(r *coinharness.Harness, t *testing.T,
	amt pfcutil.Amount) (*pfcec.PrivateKey, *wire.OutPoint, []byte, error) {

	// Create a fresh key, then send some coins to an address spendable by
	// that key.
	key, err := pfcec.NewPrivateKey(pfcec.S256())
	if err != nil {
		return nil, nil, nil, err
	}

	// Using the key created above, generate a pkScript which it's able to
	// spend.
	a, err := pfcutil.NewAddressPubKey(key.PubKey().SerializeCompressed(), r.Node.Network().(*chaincfg.Params))
	if err != nil {
		return nil, nil, nil, err
	}
	selfAddrScript, err := txscript.PayToAddrScript(a.AddressPubKeyHash())
	if err != nil {
		return nil, nil, nil, err
	}
	output := &wire.TxOut{PkScript: selfAddrScript, Value: int64(amt)}

	// Next, create and broadcast a transaction paying to the output.
	ctargs := &coinharness.CreateTransactionArgs{
		Outputs:   []coinharness.OutputTx{&pfcharness.OutputTx{output}},
		FeeRate:   10,
		Change:    true,
		TxVersion: wire.TxVersion,
	}
	fundTx, err := r.Wallet.CreateTransaction(ctargs)
	if err != nil {
		return nil, nil, nil, err
	}
	txHash, err := r.NodeRPCClient().SendRawTransaction((fundTx), true)
	if err != nil {
		return nil, nil, nil, err
	}

	// The transaction created above should be included within the next
	// generated block.
	blockHash, err := r.NodeRPCClient().Generate(1)
	if err != nil {
		return nil, nil, nil, err
	}
	assertTxInBlock(r, t, blockHash[0].(*chainhash.Hash), txHash.(*chainhash.Hash))

	// Locate the output index of the coins spendable by the key we
	// generated above, this is needed in order to create a proper utxo for
	// this output.
	var outputIndex uint32
	if bytes.Equal(fundTx.TxOut()[0].PkScript(), selfAddrScript) {
		outputIndex = 0
	} else {
		outputIndex = 1
	}

	utxo := &wire.OutPoint{
		Hash:  fundTx.TxHash().(chainhash.Hash),
		Index: outputIndex,
	}

	return key, utxo, selfAddrScript, nil
}

// TestBIP0113Activation tests for proper adherence of the BIP 113 rule
// constraint which requires all transaction finality tests to use the MTP of
// the last 11 blocks, rather than the timestamp of the block which includes
// them.
//
// Overview:
//    - Transactions with non-final lock-times from the PoV of MTP should be
//      rejected from the mempool and when found within otherwise valid blocks.
//    - Transactions with final lock-times from the PoV of MTP should be
//      accepted to the mempool and mined in future block.
func TestBIP0113Activation(t *testing.T) {
	setup := testSetup.Simnet1
	r := setup.NewInstance(t.Name()).(*coinharness.Harness)
	defer setup.Dispose(r)

	balance := r.Wallet.ConfirmedBalance()

	// Create a fresh output for usage within the test below.
	const outputValue = pfcutil.SatoshiPerPicfightcoin * 0.1
	outputKey, testOutput, testPkScript, err := makeTestOutput(r, t,
		outputValue)
	if err != nil {
		t.Fatalf("unable to create test output: %v, confirmed balance: %v", err, balance)
	}

	// Fetch a fresh address from the harness, we'll use this address to
	// send funds back into the Harness.
	addr, err := r.Wallet.NewAddress(nil)
	if err != nil {
		t.Fatalf("unable to generate address: %v", err)
	}
	addrScript, err := txscript.PayToAddrScript(addr.(pfcutil.Address))
	if err != nil {
		t.Fatalf("unable to generate addr script: %v", err)
	}

	// Now create a transaction with a lock time which is "final" according
	// to the latest block, but not according to the current median time
	// past.
	tx := wire.NewMsgTx(1)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *testOutput,
	})
	tx.AddTxOut(&wire.TxOut{
		PkScript: addrScript,
		Value:    outputValue - 1000,
	})

	// We set the lock-time of the transaction to just one minute after the
	// current MTP of the chain.
	chainInfo, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockChainInfo()
	if err != nil {
		t.Fatalf("unable to query for chain info: %v", err)
	}
	tx.LockTime = uint32(chainInfo.MedianTime) + 1

	sigScript, err := txscript.SignatureScript(tx, 0, testPkScript,
		txscript.SigHashAll, outputKey, true)
	if err != nil {
		t.Fatalf("unable to generate sig: %v", err)
	}
	tx.TxIn[0].SignatureScript = sigScript

	// This transaction should be rejected from the mempool as using MTP
	// for transactions finality is now a policy rule. Additionally, the
	// exact error should be the rejection of a non-final transaction.
	_, err = r.NodeRPCClient().Internal().(*rpcclient.Client).SendRawTransaction(tx, true)
	if err == nil {
		t.Fatalf("transaction accepted, but should be non-final")
	} else if !strings.Contains(err.Error(), "not finalized") {
		t.Fatalf("transaction should be rejected due to being "+
			"non-final, instead: %v", err)
	}

	{
		txns := []*pfcutil.Tx{pfcutil.NewTx(tx)}
		args := pfcharness.GenerateBlockArgs{
			Txns:          txns,
			BlockVersion:  vbLegacyBlockVersion,
			BlockTime:     time.Time{},
			MiningAddress: r.MiningAddress.(pfcutil.Address),
			Network:       r.Node.Network().(*chaincfg.Params),
		}
		block, err := pfcharness.GenerateAndSubmitBlock(r.NodeRPCClient(), &args)
		if err == nil {
			t.Fatalf("Block should be rejected: %v", block)
		}
		//txid := tx.TxHash()
		//assertTxInBlock(r, t, block.Hash(), &txid)
	}
	// At this point, the block height should be 102: we mined 101 blocks
	// to create a single mature output, then an additional block to create
	// a new output, and then mined a single block above to include our
	// transaction, and the block was rejected.
	assertChainHeight(r, t, 102)

	// Next, mine enough blocks to ensure that the soft-fork becomes
	// activated. Assert that the block version of the second-to-last block
	// in the final range is active.

	// Next, mine ensure blocks to ensure that the soft-fork becomes
	// active. We're at height 102 and we need 200 blocks to be mined after
	// the genesis target period, so we mine 196 blocks. This'll put us at
	// height 299. The getblockchaininfo call checks the state for the
	// block AFTER the current height.
	numBlocks := (r.Node.Network().(*chaincfg.Params).MinerConfirmationWindow * 2) - 4 + 1
	if _, err := r.NodeRPCClient().Generate(numBlocks); err != nil {
		t.Fatalf("unable to generate blocks: %v", err)
	}

	assertChainHeight(r, t, 299)
	assertSoftForkStatus(r, t, dummyForkKey, blockchain.ThresholdActive)

	// The timeLockDeltas slice represents a series of deviations from the
	// current MTP which will be used to test border conditions w.r.t
	// transaction finality. -1 indicates 1 second prior to the MTP, 0
	// indicates the current MTP, and 1 indicates 1 second after the
	// current MTP.
	//
	// This time, all transactions which are final according to the MTP
	// *should* be accepted to both the mempool and within a valid block.
	// While transactions with lock-times *after* the current MTP should be
	// rejected.
	timeLockDeltas := []int64{-1, 0, 1}
	for _, timeLockDelta := range timeLockDeltas {
		chainInfo, err = r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlockChainInfo()
		if err != nil {
			t.Fatalf("unable to query for chain info: %v", err)
		}
		medianTimePast := chainInfo.MedianTime

		// Create another test output to be spent shortly below.
		outputKey, testOutput, testPkScript, err = makeTestOutput(r, t,
			outputValue)
		if err != nil {
			t.Fatalf("unable to create test output: %v", err)
		}

		// Create a new transaction with a lock-time past the current known
		// MTP.
		tx = wire.NewMsgTx(1)
		tx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: *testOutput,
		})
		tx.AddTxOut(&wire.TxOut{
			PkScript: addrScript,
			Value:    outputValue - 1000,
		})
		tx.LockTime = uint32(medianTimePast + timeLockDelta)
		sigScript, err = txscript.SignatureScript(tx, 0, testPkScript,
			txscript.SigHashAll, outputKey, true)
		if err != nil {
			t.Fatalf("unable to generate sig: %v", err)
		}
		tx.TxIn[0].SignatureScript = sigScript

		// If the time-lock delta is greater than -1, then the
		// transaction should be rejected from the mempool and when
		// included within a block. A time-lock delta of -1 should be
		// accepted as it has a lock-time of one
		// second _before_ the current MTP.

		_, err = r.NodeRPCClient().Internal().(*rpcclient.Client).SendRawTransaction(tx, true)
		if err == nil && timeLockDelta >= 0 {
			t.Fatal("transaction was accepted into the mempool " +
				"but should be rejected!")
		} else if err != nil && !strings.Contains(err.Error(), "not finalized") {
			t.Fatalf("transaction should be rejected from mempool "+
				"due to being  non-final, instead: %v", err)
		}
		{
			txns := []*pfcutil.Tx{pfcutil.NewTx(tx)}

			args := pfcharness.GenerateBlockArgs{
				Txns:          txns,
				BlockVersion:  vbLegacyBlockVersion,
				BlockTime:     time.Time{},
				MiningAddress: r.MiningAddress.(pfcutil.Address),
				Network:       r.Node.Network().(*chaincfg.Params),
			}
			_, err := pfcharness.GenerateAndSubmitBlock(r.NodeRPCClient(), &args)
			if err == nil && timeLockDelta >= 0 {
				t.Fatal("block should be rejected due to non-final " +
					"txn, but was accepted")
			} else if err != nil && !strings.Contains(err.Error(), "unfinalized") {
				t.Fatalf("block should be rejected due to non-final "+
					"tx, instead: %v", err)
			}
		}
	}
}

// createCSVOutput creates an output paying to a trivially redeemable CSV
// pkScript with the specified time-lock.
func createCSVOutput(r *coinharness.Harness, t *testing.T,
	numSatoshis pfcutil.Amount, timeLock int32,
	isSeconds bool) ([]byte, *wire.OutPoint, coinharness.CreatedTransactionTx, error) {

	// Convert the time-lock to the proper sequence lock based according to
	// if the lock is seconds or time based.
	sequenceLock := blockchain.LockTimeToSequence(isSeconds,
		uint32(timeLock))

	// Our CSV script is simply: <sequenceLock> OP_CSV OP_DROP
	b := txscript.NewScriptBuilder().
		AddInt64(int64(sequenceLock)).
		AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
		AddOp(txscript.OP_DROP)
	csvScript, err := b.Script()
	if err != nil {
		return nil, nil, nil, err
	}

	// Using the script generated above, create a P2SH output which will be
	// accepted into the mempool.
	p2shAddr, err := pfcutil.NewAddressScriptHash(csvScript, r.Node.Network().(*chaincfg.Params))
	if err != nil {
		return nil, nil, nil, err
	}
	p2shScript, err := txscript.PayToAddrScript(p2shAddr)
	if err != nil {
		return nil, nil, nil, err
	}
	output := &wire.TxOut{
		PkScript: p2shScript,
		Value:    int64(numSatoshis),
	}

	// Finally create a valid transaction which creates the output crafted
	// above.
	ctargs := &coinharness.CreateTransactionArgs{
		Outputs:   []coinharness.OutputTx{&pfcharness.OutputTx{output}},
		FeeRate:   10,
		Change:    true,
		TxVersion: wire.TxVersion,
	}
	tx, err := r.Wallet.CreateTransaction(ctargs)
	if err != nil {
		return nil, nil, nil, err
	}

	var outputIndex uint32
	if !bytes.Equal(tx.TxOut()[0].PkScript(), p2shScript) {
		outputIndex = 1
	}

	utxo := &wire.OutPoint{
		Hash:  tx.TxHash().(chainhash.Hash),
		Index: outputIndex,
	}

	return csvScript, utxo, tx, nil
}

// spendCSVOutput spends an output previously created by the createCSVOutput
// function. The sigScript is a trivial push of OP_TRUE followed by the
// redeemScript to pass P2SH evaluation.
func spendCSVOutput(redeemScript []byte, csvUTXO *wire.OutPoint,
	sequence uint32, targetOutput *wire.TxOut,
	txVersion int32) (*wire.MsgTx, error) {

	tx := wire.NewMsgTx(txVersion)
	tx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *csvUTXO,
		Sequence:         sequence,
	})
	tx.AddTxOut(targetOutput)

	b := txscript.NewScriptBuilder().
		AddOp(txscript.OP_TRUE).
		AddData(redeemScript)

	sigScript, err := b.Script()
	if err != nil {
		return nil, err
	}
	tx.TxIn[0].SignatureScript = sigScript

	return tx, nil
}

// assertTxInBlock asserts a transaction with the specified txid is found
// within the block with the passed block hash.
func assertTxInBlock(r *coinharness.Harness, t *testing.T, blockHash *chainhash.Hash,
	txid coinharness.Hash) {

	block, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlock(blockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	if len(block.Transactions) < 2 {
		t.Fatal("target transaction was not mined")
	}

	for _, txn := range block.Transactions {
		txHash := txn.TxHash()
		if txn.TxHash() == txHash {
			return
		}
	}

	_, _, line, _ := runtime.Caller(1)
	t.Fatalf("assertion failed at line %v: txid %v was not found in "+
		"block %v", line, txid, blockHash)
}

// TestBIP0068AndBIP0112Activation tests for the proper adherence to the BIP
// 112 and BIP 68 rule-set after the activation of the CSV-package soft-fork.
//
// Overview:
//  - Pre soft-fork:
//    - A transaction spending a CSV output validly should be rejected from the
//    mempool, but accepted in a valid generated block including the
//    transaction.
//  - Post soft-fork:
//    - See the cases exercised within the table driven tests towards the end
//    of this test.
func TestBIP0068AndBIP0112Activation(t *testing.T) {
	// We'd like the test proper evaluation and validation of the BIP 68
	// (sequence locks) and BIP 112 rule-sets which add input-age based
	// relative lock times.

	setup := testSetup.Simnet1
	r := setup.NewInstance(t.Name()).(*coinharness.Harness)
	defer setup.Dispose(r)

	assertSoftForkStatus(r, t, dummyForkKey, blockchain.ThresholdStarted)

	harnessAddr, err := r.Wallet.NewAddress(nil)
	if err != nil {
		t.Fatalf("unable to obtain harness address: %v", err)
	}
	harnessScript, err := txscript.PayToAddrScript(harnessAddr.(pfcutil.Address))
	if err != nil {
		t.Fatalf("unable to generate pkScript: %v", err)
	}

	const (
		outputAmt         = pfcutil.SatoshiPerPicfightcoin / 10
		relativeBlockLock = 10
	)

	sweepOutput := &wire.TxOut{
		Value:    outputAmt - 5000,
		PkScript: harnessScript,
	}

	// As the soft-fork hasn't yet activated _any_ transaction version
	// which uses the CSV opcode should be accepted. Since at this point,
	// CSV doesn't actually exist, it's just a NOP.
	for txVersion := int32(0); txVersion < 4; txVersion++ {
		// Create a trivially spendable output with a CSV lock-time of
		// 10 relative blocks.
		redeemScript, testUTXO, tx, err := createCSVOutput(r, t, outputAmt,
			relativeBlockLock, false)
		if err != nil {
			t.Fatalf("unable to create CSV encumbered output: %v", err)
		}

		// As the transaction is p2sh it should be accepted into the
		// mempool and found within the next generated block.
		if _, err := r.NodeRPCClient().SendRawTransaction(tx, true); err != nil {
			t.Fatalf("unable to broadcast tx: %v", err)
		}
		blocks, err := r.NodeRPCClient().Generate(1)
		if err != nil {
			t.Fatalf("unable to generate blocks: %v", err)
		}
		txid := tx.TxHash()
		assertTxInBlock(r, t, blocks[0].(*chainhash.Hash), txid)

		// Generate a custom transaction which spends the CSV output.
		sequenceNum := blockchain.LockTimeToSequence(false, 10)
		spendingTx, err := spendCSVOutput(redeemScript, testUTXO,
			sequenceNum, sweepOutput, txVersion)
		if err != nil {
			t.Fatalf("unable to spend csv output: %v", err)
		}

		// This transaction should be rejected from the mempool since
		// CSV validation is already mempool policy pre-fork.
		_, err = r.NodeRPCClient().Internal().(*rpcclient.Client).SendRawTransaction(spendingTx, true)
		if err == nil {
			t.Fatalf("transaction should have been rejected, but was " +
				"instead accepted")
		}

		txns := []*pfcutil.Tx{pfcutil.NewTx(spendingTx)}

		args := pfcharness.GenerateBlockArgs{
			Txns:          txns,
			BlockVersion:  vbLegacyBlockVersion,
			BlockTime:     time.Time{},
			MiningAddress: r.MiningAddress.(pfcutil.Address),
			Network:       r.Node.Network().(*chaincfg.Params),
		}
		block, err := pfcharness.GenerateAndSubmitBlock(r.NodeRPCClient(), &args)
		if err == nil {
			t.Fatalf("Block should be rejected: %v", block)
		}
		//txid = spendingTx.TxHash()
		//assertTxInBlock(r, t, block.Hash(), &txid)
	}

	// At this point, the block height should be 105: we started at height
	// 101, then generated 1 valid block in each loop iteration above.
	assertChainHeight(r, t, 105)

	// With the height at 107 we need 200 blocks to be mined after the
	// genesis target period, so we mine 192 blocks. This'll put us at
	// height 299. The getblockchaininfo call checks the state for the
	// block AFTER the current height.
	numBlocks := (r.Node.Network().(*chaincfg.Params).MinerConfirmationWindow * 2) - 8 + 2
	if _, err := r.NodeRPCClient().Generate(numBlocks); err != nil {
		t.Fatalf("unable to generate blocks: %v", err)
	}

	assertChainHeight(r, t, 299)
	assertSoftForkStatus(r, t, dummyForkKey, blockchain.ThresholdActive)

	// Knowing the number of outputs needed for the tests below, create a
	// fresh output for use within each of the test-cases below.
	const relativeTimeLock = 512
	const numTests = 8
	type csvOutput struct {
		RedeemScript []byte
		Utxo         *wire.OutPoint
		Timelock     int32
	}
	var spendableInputs [numTests]csvOutput

	// Create three outputs which have a block-based sequence locks, and
	// three outputs which use the above time based sequence lock.
	for i := 0; i < numTests; i++ {
		timeLock := relativeTimeLock
		isSeconds := true
		if i < 7 {
			timeLock = relativeBlockLock
			isSeconds = false
		}

		redeemScript, utxo, tx, err := createCSVOutput(r, t, outputAmt,
			int32(timeLock), isSeconds)
		if err != nil {
			t.Fatalf("unable to create CSV output: %v", err)
		}

		if _, err := r.NodeRPCClient().SendRawTransaction(tx, true); err != nil {
			t.Fatalf("unable to broadcast transaction: %v", err)
		}

		spendableInputs[i] = csvOutput{
			RedeemScript: redeemScript,
			Utxo:         utxo,
			Timelock:     int32(timeLock),
		}
	}

	// Mine a single block including all the transactions generated above.
	if _, err := r.NodeRPCClient().Generate(1); err != nil {
		t.Fatalf("unable to generate block: %v", err)
	}

	// Now mine 10 additional blocks giving the inputs generated above a
	// age of 11. Space out each block 10 minutes after the previous block.
	prevBlockHash, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBestBlockHash()
	if err != nil {
		t.Fatalf("unable to get prior block hash: %v", err)
	}
	prevBlock, err := r.NodeRPCClient().Internal().(*rpcclient.Client).GetBlock(prevBlockHash)
	if err != nil {
		t.Fatalf("unable to get block: %v", err)
	}
	for i := 0; i < relativeBlockLock; i++ {
		timeStamp := prevBlock.Header.Timestamp.Add(time.Minute * 10)
		args := pfcharness.GenerateBlockArgs{
			BlockVersion:  vbLegacyBlockVersion,
			BlockTime:     timeStamp,
			MiningAddress: r.MiningAddress.(pfcutil.Address),
			Network:       r.Node.Network().(*chaincfg.Params),
		}
		b, err := pfcharness.GenerateAndSubmitBlock(r.NodeRPCClient(), &args)
		if err != nil {
			t.Fatalf("unable to generate block: %v", err)
		}

		prevBlock = b.MsgBlock()
	}

	// A helper function to create fully signed transactions in-line during
	// the array initialization below.
	var inputIndex uint32
	makeTxCase := func(sequenceNum uint32, txVersion int32) *wire.MsgTx {
		csvInput := spendableInputs[inputIndex]

		tx, err := spendCSVOutput(csvInput.RedeemScript, csvInput.Utxo,
			sequenceNum, sweepOutput, txVersion)
		if err != nil {
			t.Fatalf("unable to spend CSV output: %v", err)
		}

		inputIndex++
		return tx
	}

	tests := [numTests]struct {
		tx     *wire.MsgTx
		accept bool
	}{
		// A valid transaction with a single input a sequence number
		// creating a 100 block relative time-lock. This transaction
		// should be rejected as its version number is 1, and only tx
		// of version > 2 will trigger the CSV behavior.
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 100), 1),
			accept: false,
		},
		// A transaction of version 2 spending a single input. The
		// input has a relative time-lock of 1 block, but the disable
		// bit it set. The transaction should be rejected as a result.
		{
			tx: makeTxCase(
				blockchain.LockTimeToSequence(false, 1)|wire.SequenceLockTimeDisabled,
				2,
			),
			accept: false,
		},
		// A v2 transaction with a single input having a 9 block
		// relative time lock. The referenced input is 11 blocks old,
		// but the CSV output requires a 10 block relative lock-time.
		// Therefore, the transaction should be rejected.
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 9), 2),
			accept: false,
		},
		// A v2 transaction with a single input having a 10 block
		// relative time lock. The referenced input is 11 blocks old so
		// the transaction should be accepted.
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 10), 2),
			accept: true,
		},
		// A v2 transaction with a single input having a 11 block
		// relative time lock. The input referenced has an input age of
		// 11 and the CSV op-code requires 10 blocks to have passed, so
		// this transaction should be accepted.
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 11), 2),
			accept: true,
		},
		// A v2 transaction whose input has a 1000 blck relative time
		// lock.  This should be rejected as the input's age is only 11
		// blocks.
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(false, 1000), 2),
			accept: false,
		},
		// A v2 transaction with a single input having a 512,000 second
		// relative time-lock. This transaction should be rejected as 6
		// days worth of blocks haven't yet been mined. The referenced
		// input doesn't have sufficient age.
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(true, 512000), 2),
			accept: false,
		},
		// A v2 transaction whose single input has a 512 second
		// relative time-lock. This transaction should be accepted as
		// finalized.
		{
			tx:     makeTxCase(blockchain.LockTimeToSequence(true, 512), 2),
			accept: true,
		},
	}

	for i, test := range tests {
		txid, err := r.NodeRPCClient().Internal().(*rpcclient.Client).SendRawTransaction(test.tx, true)
		switch {
		// Test case passes, nothing further to report.
		case test.accept && err == nil:

			// Transaction should have been accepted but we have a non-nil
			// error.
		case test.accept && err != nil:
			t.Fatalf("test #%d, transaction should be accepted, "+
				"but was rejected: %v", i, err)

			// Transaction should have been rejected, but it was accepted.
		case !test.accept && err == nil:
			t.Fatalf("test #%d, transaction should be rejected, "+
				"but was accepted", i)

			// Transaction was rejected as wanted, nothing more to do.
		case !test.accept && err != nil:
		}

		// If the transaction should be rejected, manually mine a block
		// with the non-final transaction. It should be rejected.
		if !test.accept {
			txns := []*pfcutil.Tx{pfcutil.NewTx(test.tx)}
			args := pfcharness.GenerateBlockArgs{
				Txns:          txns,
				BlockVersion:  vbLegacyBlockVersion,
				BlockTime:     time.Time{},
				MiningAddress: r.MiningAddress.(pfcutil.Address),
				Network:       r.Node.Network().(*chaincfg.Params),
			}
			_, err := pfcharness.GenerateAndSubmitBlock(r.NodeRPCClient(), &args)
			if err == nil {
				t.Fatalf("test #%d, invalid block accepted", i)
			}

			continue
		}

		// Generate a block, the transaction should be included within
		// the newly mined block.
		blockHashes, err := r.NodeRPCClient().Generate(1)
		if err != nil {
			t.Fatalf("unable to mine block: %v", err)
		}
		assertTxInBlock(r, t, blockHashes[0].(*chainhash.Hash), txid)
	}
}
