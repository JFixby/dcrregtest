package decred_regression_testing

import (
	"github.com/jfixby/pin"
	"github.com/picfight/pfcd_builder/fileops"
	"testing"
)

func TestFindDCR(t *testing.T) {
	path := fileops.Abs("../../decred/dcrd")
	pin.D("path", path)
}
