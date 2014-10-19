package reg

// xlReg_go/cluster_member_test.go

import (
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	//xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	. "gopkg.in/check.v1"
)

// XXX THIS IS CONFUSED.  A RegCluster is something that appears on
// the server side.  It contains a set of ClientInfo objects.  This
// test was written when ClientInfo was the same as MemberInfo.  The
// problem is that ClientInfo objects have no serialization routines.

func (s *XLSuite) TestClusterMemberSerialization(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_CLUSTER_MEMBER_SERIALIZATION")
	}
	rng := xr.MakeSimpleRNG()
	var err error

	// Generate a random cluster
	epCount := uint32(1 + rng.Intn(3)) // so from 1 to 3
	size := uint32(2 + rng.Intn(6))    // so from 2 to 7
	cl := s.makeACluster(c, rng, epCount, size)

	cl0EpCount := uint32(len(cl.Members[0].MyEnds))
	c.Assert(cl0EpCount, Equals, epCount) // XXX FAILS: 0, 1

	// We are going to overwrite cluster member zero's attributes
	// with those of the new cluster member.

	// Make Node copy.
	namesInUse := make(map[string]bool)
	myNode, myCkPriv, mySkPriv := s.makeHostAndKeys(c, rng, namesInUse)
	for i := uint32(0); i < epCount; i++ {
		var (
			ep  xt.EndPointI
			ndx int
		)
		ep, err = xt.NewTcpEndPoint(cl.Members[0].MyEnds[i])
		c.Assert(err, IsNil)
		ndx, err = myNode.AddEndPoint(ep)
		c.Assert(err, IsNil)
		c.Assert(uint32(ndx), Equals, i)
	}
	c.Assert(uint32(myNode.SizeEndPoints()), Equals, epCount)

	myAttrs := cl.Members[0].Attrs

	var myCtors []xt.ConnectorI
	for i := uint32(0); i < epCount; i++ {
		var ctor xt.ConnectorI
		ctor, err = xt.NewTcpConnector(myNode.GetEndPoint(int(i)))
		c.Assert(err, IsNil)
		myCtors = append(myCtors, ctor)
	}
	meAsPeer, err := xn.NewPeer(myNode.GetName(), myNode.GetNodeID(),
		&myCkPriv.PublicKey, &mySkPriv.PublicKey, nil, myCtors)

	// XXX NEEDS FIXING FROM HERE
	_, _ = myAttrs, meAsPeer

	//myMemberInfo, err := NewMemberInfo(myAttrs, meAsPeer)
	//c.Assert(err, IsNil)
	//// overwrite member 0
	//cl.Members[0] = myMemberInfo

	//myClusterID, err := xi.New(cl.ID)
	//c.Assert(err, IsNil)

	//cm := &ClusterMember{
	//	Attrs:        myAttrs,
	//	ClusterName:  cl.Name,
	//	ClusterAttrs: cl.Attrs,
	//	ClusterID:    myClusterID,
	//	ClusterSize:  cl.MaxSize(),
	//	SelfIndex:    uint32(0),
	//	Members:      cl.Members, // []*MemberInfo
	//	EPCount:      epCount,
	//	Node:         *myNode,
	//}

	//// simplest test of Equal()
	//c.Assert(cm.Equal(cm), Equals, true)

	//// Serialize it
	//serialized := cm.String()

	//// Reverse the serialization
	//deserialized, rest, err := ParseClusterMember(serialized)
	//c.Assert(err, IsNil)
	//c.Assert(deserialized, Not(IsNil))
	//c.Assert(len(rest), Equals, 0)

	//// Verify that the deserialized cluster is identical to the original
	//// First version:
	//c.Assert(deserialized.Equal(cm), Equals, true)

	//// Second version of identity test:
	//serialized2 := deserialized.String()
	//c.Assert(serialized2, Equals, serialized)
}
