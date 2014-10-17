package reg

// xlReg_go/reg_cred_test.go

import (
	// "crypto/rsa"
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
	. "gopkg.in/check.v1"
)

func (s *XLSuite) makeRegCred(c *C, rng *xr.PRNG) (rc *RegCred) {

	name := rng.NextFileName(8)
	nodeID, _ := xi.New(nil)
	node, err := xn.NewNew(name, nodeID, "") // "" is LFS
	c.Assert(err, IsNil)
	epCount := rng.Intn(4)
	var e []xt.EndPointI
	for i := 0; i < epCount; i++ {
		port := 1024 + rng.Intn(256*256-1024)
		strAddr := fmt.Sprintf("127.0.0.1:%d", port)
		ep, err := xt.NewTcpEndPoint(strAddr)
		c.Assert(err, IsNil)
		e = append(e, ep)
	}
	version := xu.DecimalVersion(uint32(rng.Int31()))
	rc = &RegCred{
		Name:        name,
		ID:          nodeID,
		CommsPubKey: node.GetCommsPublicKey(),
		SigPubKey:   node.GetSigPublicKey(),
		EndPoints:   e,
		Version:     version,
	}
	return
}
func (s *XLSuite) TestRegCred(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_REG_CRED_TEST")
	}
	rng := xr.MakeSimpleRNG()
	for i := 0; i < 4; i++ {
		rc := s.makeRegCred(c, rng)
		serialized := rc.String()
		backAgain, err := ParseRegCred(serialized)
		c.Assert(err, IsNil)
		serialized2 := backAgain.String()
		c.Assert(serialized2, Equals, serialized)
	}
}
