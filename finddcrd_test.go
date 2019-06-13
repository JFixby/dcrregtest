package dcrregtest

import (
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/fileops"
	"testing"
)

func TestFindDCR(t *testing.T) {
	path := fileops.Abs("../../decred/dcrd")
	pin.D("path", path)
}
