package reg

// xlReg_go/cnxHandler_test.go

import (
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	. "gopkg.in/check.v1"
)

var _ = fmt.Print

func (s *XLSuite) TestCnxHandler(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_CNX_HANDLER")
	}
	rng := xr.MakeSimpleRNG()

	_ = rng

	// XXX STUB XXX
}
