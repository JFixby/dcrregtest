// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testnode

import (
	"github.com/jfixby/cointest"
	"github.com/jfixby/dcrregtest"
	"github.com/jfixby/dcrregtest/consolenode"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
	"path/filepath"
)

// NodeFactory produces a new ConsoleNode-instance upon request
type NodeFactory struct {
	// NodeExecutablePathProvider returns path to the dcrd executable
	NodeExecutablePathProvider commandline.ExecutablePathProvider
	ConsoleCommandCook         DcrdConsoleCommandCook
	RPCClientFactory           dcrregtest.DcrRPCClientFactory
}

// NewNode creates and returns a fully initialized instance of the ConsoleNode.
func (factory *NodeFactory) NewNode(config *cointest.TestNodeConfig) cointest.TestNode {
	pin.AssertNotNil("WorkingDir", config.WorkingDir)
	pin.AssertNotEmpty("WorkingDir", config.WorkingDir)

	args := &consolenode.NewConsoleNodeArgs{
		ClientFac:                  &factory.RPCClientFactory,
		ConsoleCommandCook:         &factory.ConsoleCommandCook,
		NodeExecutablePathProvider: factory.NodeExecutablePathProvider,
		RpcUser:                    "user",
		RpcPass:                    "pass",
		AppDir:                     filepath.Join(config.WorkingDir, "dcrd"),
		P2PHost:                    config.P2PHost,
		P2PPort:                    config.P2PPort,
		NodeRPCHost:                config.NodeRPCHost,
		NodeRPCPort:                config.NodeRPCPort,
		ActiveNet:                  config.ActiveNet,
	}

	return consolenode.NewConsoleNode(args)
}
