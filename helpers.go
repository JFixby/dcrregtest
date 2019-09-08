// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dcrregtest

import (
	"fmt"
	"github.com/decred/dcrd/rpcclient"
	"github.com/jfixby/coinharness"
	"strconv"
	"testing"
)

// Create a test chain with the desired number of mature coinbase outputs
func generateTestChain(numToGenerate int64, node *rpcclient.Client) error {
	fmt.Printf("Generating %v blocks...\n", numToGenerate)
	_, err := node.Generate(uint32(numToGenerate))
	if err != nil {
		return err
	}
	fmt.Println("Block generation complete.")
	return nil
}

func assertConnectedTo(t *testing.T, nodeA *coinharness.Harness, nodeB *coinharness.Harness) {
	nodeAPeers, err := nodeA.NodeRPCClient().Internal().(*rpcclient.Client).GetPeerInfo()
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

// Waits for wallet to sync to the target height
func syncWalletTo(rpcClient *rpcclient.Client, desiredHeight int64) (int64, error) {
	var count int64
	var err error
	for count != desiredHeight {
		//rpctest.Sleep(100)
		count, err = rpcClient.GetBlockCount()
		if err != nil {
			return -1, err
		}
		fmt.Println("   sync to: " + strconv.FormatInt(count, 10))
	}
	return count, nil
}
