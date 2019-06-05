// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dcrregtest

import (
	"github.com/decred/dcrd/rpcclient"
	"github.com/jfixby/cointest"
)

type DcrRPCClientFactory struct {
}

func (f *DcrRPCClientFactory) NewRPCConnection(config cointest.RPCConnectionConfig, handlers cointest.RPCClientNotificationHandlers) (cointest.RPCClient, error) {
	var h *rpcclient.NotificationHandlers
	if handlers != nil {
		h = handlers.
		(*rpcclient.NotificationHandlers)
	}
	return rpcclient.New(config.(*rpcclient.ConnConfig), h)
}
