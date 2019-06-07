// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dcrregtest

import (
	"fmt"
	"github.com/jfixby/cointest"
	"github.com/jfixby/dcrtest/memwallet"
	"github.com/jfixby/dcrtest/nodecls"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
	"github.com/jfixby/pin/gobuilder"
	"github.com/picfight/pfcd_builder/fileops"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/decred/dcrd/chaincfg"
)

// Default harness name
const mainHarnessName = "main"

// SimpleTestSetup harbours:
// - rpctest setup
// - csf-fork test setup
// - and bip0009 test setup
type SimpleTestSetup struct {
	// harnessPool stores and manages harnesses
	// multiple harness instances may be run concurrently, to allow for testing
	// complex scenarios involving multiple nodes.
	harnessPool *pin.Pool

	// Regnet25 creates a regnet test harness
	// with 25 mature outputs.
	Regnet25 *ChainWithMatureOutputsSpawner

	// Regnet5 creates a regnet test harness
	// with 5 mature outputs.
	Regnet5 *ChainWithMatureOutputsSpawner

	// Regnet1 creates a regnet test harness
	// with 1 mature output.
	Regnet1 *ChainWithMatureOutputsSpawner

	// Simnet1 creates a simnet test harness
	// with 1 mature output.
	Simnet1 *ChainWithMatureOutputsSpawner

	// Regnet0 creates a regnet test harness
	// with only the genesis block.
	Regnet0 *ChainWithMatureOutputsSpawner

	// Simnet0 creates a simnet test harness
	// with only the genesis block.
	Simnet0 *ChainWithMatureOutputsSpawner

	// ConsoleNodeFactory produces a new TestNode instance upon request
	NodeFactory cointest.TestNodeFactory

	// WalletFactory produces a new TestWallet instance upon request
	WalletFactory cointest.TestWalletFactory

	// WorkingDir defines test setup working dir
	WorkingDir *pin.TempDirHandler
}

// TearDown all harnesses in test Pool.
// This includes removing all temporary directories,
// and shutting down any created processes.
func (setup *SimpleTestSetup) TearDown() {
	setup.harnessPool.DisposeAll()
	//setup.nodeGoBuilder.Dispose()
	setup.WorkingDir.Dispose()
}

// Setup deploys this test setup
func Setup() *SimpleTestSetup {
	setup := &SimpleTestSetup{
		WalletFactory: &memwallet.WalletFactory{},
		//Network:       &chaincfg.RegNetParams,
		WorkingDir: pin.NewTempDir(setupWorkingDir(), "simpleregtest").MakeDir(),
	}

	dcrdEXE := &commandline.ExplicitExecutablePathString{PathString: "../../../decred/dcrd/dcrd.exe"}

	//buildName := "dcrd"
	//nodeProjectGoPath := findDCRDProjectPath()

	//setup.nodeGoBuilder = setupBuild(buildName, setup.WorkingDir.Path(), nodeProjectGoPath)
	setup.NodeFactory = &nodecls.ConsoleNodeFactory{
		NodeExecutablePathProvider: dcrdEXE,
	}
	//setup.nodeGoBuilder.Build()

	portManager := &LazyPortManager{
		BasePort: 20000,
		offset:   0,
	}

	// Deploy harness spawner with generated
	// test chain of 25 mature outputs
	setup.Regnet25 = &ChainWithMatureOutputsSpawner{
		WorkingDir:        setup.WorkingDir.Path(),
		DebugNodeOutput:   true,
		DebugWalletOutput: true,
		NumMatureOutputs:  25,
		NetPortManager:    portManager,
		WalletFactory:     setup.WalletFactory,
		NodeFactory:       setup.NodeFactory,
		ActiveNet:         &chaincfg.RegNetParams,
	}

	// Deploy harness spawner with generated
	// test chain of 5 mature outputs
	setup.Regnet5 = &ChainWithMatureOutputsSpawner{
		WorkingDir:        setup.WorkingDir.Path(),
		DebugNodeOutput:   true,
		DebugWalletOutput: true,
		NumMatureOutputs:  5,
		NetPortManager:    portManager,
		WalletFactory:     setup.WalletFactory,
		NodeFactory:       setup.NodeFactory,
		ActiveNet:         &chaincfg.RegNetParams,
	}

	setup.Regnet1 = &ChainWithMatureOutputsSpawner{
		WorkingDir:        setup.WorkingDir.Path(),
		DebugNodeOutput:   true,
		DebugWalletOutput: true,
		NumMatureOutputs:  1,
		NetPortManager:    portManager,
		WalletFactory:     setup.WalletFactory,
		NodeFactory:       setup.NodeFactory,
		ActiveNet:         &chaincfg.RegNetParams,
		NodeStartExtraArguments: map[string]interface{}{
			"rejectnonstd": commandline.NoArgumentValue,
		},
	}

	setup.Simnet1 = &ChainWithMatureOutputsSpawner{
		WorkingDir:        setup.WorkingDir.Path(),
		DebugNodeOutput:   true,
		DebugWalletOutput: true,
		NumMatureOutputs:  1,
		NetPortManager:    portManager,
		WalletFactory:     setup.WalletFactory,
		NodeFactory:       setup.NodeFactory,
		ActiveNet:         &chaincfg.SimNetParams,
		NodeStartExtraArguments: map[string]interface{}{
			"rejectnonstd": commandline.NoArgumentValue,
		},
	}

	// Deploy harness spawner with empty test chain
	setup.Regnet0 = &ChainWithMatureOutputsSpawner{
		WorkingDir:        setup.WorkingDir.Path(),
		DebugNodeOutput:   false,
		DebugWalletOutput: false,
		NumMatureOutputs:  0,
		NetPortManager:    portManager,
		WalletFactory:     setup.WalletFactory,
		NodeFactory:       setup.NodeFactory,
		ActiveNet:         &chaincfg.RegNetParams,
	}
	// Deploy harness spawner with empty test chain
	setup.Simnet0 = &ChainWithMatureOutputsSpawner{
		WorkingDir:        setup.WorkingDir.Path(),
		DebugNodeOutput:   false,
		DebugWalletOutput: false,
		NumMatureOutputs:  0,
		NetPortManager:    portManager,
		WalletFactory:     setup.WalletFactory,
		NodeFactory:       setup.NodeFactory,
		ActiveNet:         &chaincfg.SimNetParams,
	}

	setup.harnessPool = pin.NewPool(setup.Regnet25)

	return setup
}

func findDCRDProjectPath() string {
	path := fileops.Abs("../../../decred/dcrd")
	pin.D("path", path)
	return path
}

func setupWorkingDir() string {
	testWorkingDir, err := ioutil.TempDir("", "integrationtest")
	if err != nil {
		fmt.Println("Unable to create working dir: ", err)
		os.Exit(-1)
	}
	return testWorkingDir
}

func setupBuild(buildName string, workingDir string, nodeProjectGoPath string) *gobuilder.GoBuider {

	tempBinDir := filepath.Join(workingDir, "bin")
	pin.MakeDirs(tempBinDir)

	nodeGoBuilder := &gobuilder.GoBuider{
		GoProjectPath:    nodeProjectGoPath,
		OutputFolderPath: tempBinDir,
		BuildFileName:    buildName,
	}
	return nodeGoBuilder
}
