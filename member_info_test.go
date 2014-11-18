package reg

// xlReg_go/member_info_test.go

import (
	"fmt"
	//xcl "github.com/jddixon/xlCluster_go"
	xr "github.com/jddixon/rnglib_go"
	. "gopkg.in/check.v1"
)

func (s *XLSuite) TestMemberInfoAndTokens(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_MEMBER_INFO_AND_TOKENS")
	}
	rng := xr.MakeSimpleRNG()

	// Generate a random cluster member
	cm := s.makeAMemberInfo(c, rng)

	token, err := TokenFromMemberInfo(cm)
	c.Assert(err, IsNil)

	cm2, err := NewMemberInfoFromToken(token)
	c.Assert(err, IsNil)
	c.Assert(cm.Equal(cm2), Equals, true)
}
