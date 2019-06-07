// Copyright (c) 2018 The btcsuite developers
// Copyright (c) 2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package dcrregtest

import (
	"fmt"
	"github.com/decred/dcrd/rpcclient"
	"github.com/jfixby/cointest"
	"github.com/jfixby/pin"
	"io/ioutil"
)

type DcrRPCClientFactory struct {
}

func (f *DcrRPCClientFactory) NewRPCConnection(config cointest.RPCConnectionConfig, handlers cointest.RPCClientNotificationHandlers) (cointest.RPCClient, error) {
	var h *rpcclient.NotificationHandlers
	if handlers != nil {
		h = handlers.
		(*rpcclient.NotificationHandlers)
	}

	file := config.CertificateFile
	fmt.Println("reading: " + file)
	cert, err := ioutil.ReadFile(file)
	pin.CheckTestSetupMalfunction(err)

	cfg := &rpcclient.ConnConfig{
		Host:                 config.Host,
		Endpoint:             config.Endpoint,
		User:                 config.User,
		Pass:                 config.Pass,
		Certificates:         cert,
		DisableAutoReconnect: true,
		HTTPPostMode:         false,
	}

	return rpcclient.New(cfg, h)
}
