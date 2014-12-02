package reg

// xlReg_go_node_test.go

import (
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	. "gopkg.in/check.v1"
	"time"
)

func (s *XLSuite) TestRegNode(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_REG_NODE")
	}
}

// TEST SERIALIZATION ///////////////////////////////////////////////
func (s *XLSuite) TestRegNodeSerialization(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_REG_NODE_SERIALIZATION")
	}
	rng := xr.MakeSimpleRNG()

	namesInUse := make(map[string]bool)
	node, ckPriv, skPriv := s.makeHostAndKeys(c, rng, namesInUse)

	// This assigns an endPoint in 127.0.0.1 to the node.
	s.makeALocalEndPoint(c, node)
	c.Assert(node.SizeEndPoints(), Equals, 1)
	err := node.OpenAcc()
	c.Assert(err, IsNil)
	c.Assert(node.SizeAcceptors(), Equals, 1)
	regNode, err := NewRegNode(node, ckPriv, skPriv)
	c.Assert(err, IsNil)

	serialized := regNode.String()

	// We can't deserialize the node - it contains a live acceptor
	// at the same endPoint.
	//for i := 0; i < regNode.SizeAcceptors(); i++ {
	//	regNode.GetAcceptor(i).Close()
	//}
	err = regNode.CloseAcc()
	c.Assert(err, IsNil)

	// the Node version of this fails if sleep is say 10ms
	time.Sleep(70 * time.Millisecond)

	backAgain, rest, err := ParseRegNode(serialized)

	// DEBUG
	if len(rest) > 0 {
		for i := 0; i < len(rest); i++ {
			fmt.Printf("REST: %s\n", rest[i])
		}
	}
	// END
	c.Assert(err, IsNil)
	c.Assert(len(rest), Equals, 0)

	reserialized := backAgain.String()
	c.Assert(reserialized, Equals, serialized)
}
