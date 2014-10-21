package reg

// xlReg_go/clientInfo_test.go

import (
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	. "gopkg.in/check.v1"
)

func (s *XLSuite) TestCISerialization(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_CI_SERIALIZATION")
	}
	rng := xr.MakeSimpleRNG()
	epCount := uint32(1 + rng.Intn(3))

	// Generate a random cluster member
	cm := s.makeAClientInfo(c, rng, epCount)

	// Serialize it
	serialized := cm.String()

	// Reverse the serialization
	deserialized, rest, err := ParseClientInfo(serialized)
	c.Assert(err, IsNil)
	c.Assert(len(rest), Equals, 0)

	// Verify that the deserialized member is identical to the original
	c.Assert(deserialized.Equal(cm), Equals, true)
}

func (s *XLSuite) TestClientInfoAndTokens(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_CLIENT_INFO_AND_TOKENS")
	}
	rng := xr.MakeSimpleRNG()
	epCount := uint32(1 + rng.Intn(3))

	// Generate a random cluster member
	cm := s.makeAClientInfo(c, rng, epCount)

	token, err := cm.Token()
	c.Assert(err, IsNil)

	cm2, err := NewClientInfoFromToken(token)
	c.Assert(err, IsNil)
	c.Assert(cm.Equal(cm2), Equals, true)
}
