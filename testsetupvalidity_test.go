package dcrregtest

import (
	"fmt"
	"github.com/decred/dcrd/dcrutil"
	"testing"
)

func TestSetupValidity(t *testing.T) {
	coins50 := dcrutil.Amount(50 /*DCR*/ * 1e8)
	stringVal := fmt.Sprintf("%v", coins50)
	expectedStringVal := "50 DCR"
	//pin.D("stringVal", stringVal)
	if expectedStringVal != stringVal {
		t.Fatalf("Incorrect coin: "+
			"expected %v, got %v", expectedStringVal, stringVal)
	}
}
