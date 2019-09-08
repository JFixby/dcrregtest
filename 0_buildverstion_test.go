package dcrregtest

import (
	"fmt"
	"github.com/jfixby/coinharness"
	"testing"
)

func TestBuildVerstion(t *testing.T) {
	//if testing.Short() {
	//	t.Skip("Skipping RPC harness tests in short mode")
	//}
	pool := testSetup.Regnet0
	r := pool.NewInstance(t.Name()).(*coinharness.Harness)
	defer pool.Dispose(r)
	// Create a new block connecting to the current tip.
	result, err := r.NodeRPCClient().GetBuildVersion()
	EXPECTED := "decred does not support this feature (GetBuildVersion)"
	if fmt.Sprint(err) != EXPECTED {
		t.Fatalf("GetBuildVersion result: <%v> <%v>, expected <%v>", result, err, EXPECTED)
	}

}
