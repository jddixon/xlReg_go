package reg

// xlReg_go/cluster_member_test.go

import (
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	//xi "github.com/jddixon/xlNodeID_go"
	//xn "github.com/jddixon/xlNode_go"
	//xt "github.com/jddixon/xlTransport_go"
	. "gopkg.in/check.v1"
)

func (s *XLSuite) TestClusterMemberSerialization(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_CLUSTER_MEMBER_SERIALIZATION")
	}
	rng := xr.MakeSimpleRNG()
	var err error

	// Generate a random test cluster
	name := rng.NextFileName(8)
	nodeID := s.makeANodeID(c, rng)
	attrs := uint64(rng.Int63())
	size := uint32(2 + rng.Intn(6))    // so from 2 to 7
	epCount := uint32(1 + rng.Intn(3)) // so from 1 to 3

	tc, err := NewTestCluster(name, nodeID, attrs, size, epCount)
	c.Assert(err, IsNil)
	c.Assert(tc, NotNil)

	// populate the test cluster ////////////////////////////////////

	// add members ////////////////////////////////////////
	namesInUse := make(map[string]bool)
	for i := uint32(0); i < size; i++ {
		node, ckPriv, skPriv := s.makeHostAndKeys(c, rng, namesInUse)
		_, _ = ckPriv, skPriv
		attrs := uint64(rng.Int63())
		var member *ClusterMember
		member, err = tc.AddToCluster(node, attrs)
		c.Assert(err, IsNil)
		c.Assert(member, NotNil)
		c.Assert(member.SelfIndex, Equals, i)

		// WORKING HERE
	}
	// add MemberInfo to each cluster member //////////////
	// XXX STUB XXX

	// verify indexes (ClMembersByName, ClMembersByID /////
	// XXX STUB XXX

	// verify that each member's MemberInfo array is correct
	// XXX STUB XXX

	// XXX NEEDS FIXING OR JUST JUNK FROM HERE XXX //////////////////

	//cl0EpCount := uint32(len(cl.Members[0].MyEnds))
	//c.Assert(cl0EpCount, Equals, epCount)

	//// We are going to overwrite cluster member zero's attributes
	//// with those of the new cluster member.

	//// Make Node copy.
	//namesInUse := make(map[string]bool)
	//myNode, myCkPriv, mySkPriv := s.makeHostAndKeys(c, rng, namesInUse)
	//for i := uint32(0); i < epCount; i++ {
	//	var (
	//		ep  xt.EndPointI
	//		ndx int
	//	)
	//	ep, err = xt.NewTcpEndPoint(cl.Members[0].MyEnds[i])
	//	c.Assert(err, IsNil)
	//	ndx, err = myNode.AddEndPoint(ep)
	//	c.Assert(err, IsNil)
	//	c.Assert(uint32(ndx), Equals, i)
	//}
	//c.Assert(uint32(myNode.SizeEndPoints()), Equals, epCount)

	//myAttrs := cl.Members[0].Attrs

	//var myCtors []xt.ConnectorI
	//for i := uint32(0); i < epCount; i++ {
	//	var ctor xt.ConnectorI
	//	ctor, err = xt.NewTcpConnector(myNode.GetEndPoint(int(i)))
	//	c.Assert(err, IsNil)
	//	myCtors = append(myCtors, ctor)
	//}
	//meAsPeer, err := xn.NewPeer(myNode.GetName(), myNode.GetNodeID(),
	//	&myCkPriv.PublicKey, &mySkPriv.PublicKey, nil, myCtors)

	//_, _ = myAttrs, meAsPeer

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
