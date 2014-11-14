package reg

// xlReg_go/cluster_member_test.go

import (
	"crypto/rsa"
	"fmt"
	xh "github.com/jddixon/hamt_go"
	xr "github.com/jddixon/rnglib_go"
	//xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	. "gopkg.in/check.v1"
)

// This test creates a test cluster using NewTestCluster() but does
// NOT use the xlReg registry.
//
func (s *XLSuite) TestClusterMemberSerialization(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_CLUSTER_MEMBER_SERIALIZATION")
	}
	rng := xr.MakeSimpleRNG()
	var (
		err              error
		members          []*ClusterMember
		ckPrivs, skPrivs []*rsa.PrivateKey
	)

	// defer closing each member's acceptors, unless nil
	defer func() {
		if members != nil {
			for i := 0; i < len(members); i++ {
				node := members[i].Node
				count := node.SizeAcceptors()
				for j := 0; j < count; j++ {
					acc := node.GetAcceptor(j)
					if acc != nil {
						acc.Close()
					}
				}
			}
		}
	}()

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
		ckPrivs = append(ckPrivs, ckPriv)
		skPrivs = append(skPrivs, skPriv)

		// add epCount endPoints to the node, creating acceptors
		for j := uint32(0); j < epCount; j++ {
			var (
				ep  *xt.TcpEndPoint
				ndx int
			)
			ep, err = xt.NewTcpEndPoint("127.0.0.1:0")
			c.Assert(err, IsNil)
			ndx, err = node.AddEndPoint(ep)
			c.Assert(err, IsNil)
			c.Assert(ndx, Equals, int(j))
		}
		c.Assert(node.SizeEndPoints(), Equals, int(epCount))

		attrs := uint64(rng.Int63())
		var member *ClusterMember
		member, err = tc.AddToCluster(node, attrs)
		c.Assert(err, IsNil)
		c.Assert(member, NotNil)
		c.Assert(member.SelfIndex, Equals, i)
		members = append(members, member)
	}
	// add MemberInfo to each cluster member //////////////
	var memberInfos []*MemberInfo
	for i := uint32(0); i < size; i++ {
		var peer *xn.Peer
		peer, err = xn.NewPeer(members[i].GetName(), members[i].GetNodeID(),
			&ckPrivs[i].PublicKey, &skPrivs[i].PublicKey,
			nil, nil) // overlays, connectors
		c.Assert(err, IsNil)
		c.Assert(peer, NotNil)
		attrs := uint64(rng.Int63())
		var mi *MemberInfo
		mi, err = NewMemberInfo(attrs, peer)
		c.Assert(err, IsNil)
		c.Assert(mi, NotNil)
		memberInfos = append(memberInfos, mi)
	}
	for i := uint32(0); i < size; i++ {
		members[i].Members = memberInfos
	}

	// add peers to each ClusterMember Node ///////////////

	// XXX STUB

	// add connectors to peers

	// XXX STUB

	// verify indexes (ClMembersByName, ClMembersByID /////
	for i := uint32(0); i < size; i++ {
		name := members[i].GetName()
		nodeID := members[i].GetNodeID()

		memberByName := tc.ClMembersByName[name]
		c.Assert(members[i].Equal(memberByName), Equals, true)

		var bKey xh.BytesKey
		bKey, err = xh.NewBytesKey(nodeID.Value())
		c.Assert(err, IsNil)
		c.Assert(bKey, NotNil)

		byID, err := tc.ClMembersByID.Find(bKey)
		c.Assert(err, IsNil)
		c.Assert(byID, NotNil)
		memberByID, ok := byID.(*ClusterMember)
		c.Assert(ok, Equals, true)
		c.Assert(members[i].Equal(memberByID), Equals, true)
	}
	// XXX NEEDS FIXING FROM HERE XXX ///////////////////////////////

	//var myCtors []xt.ConnectorI
	//for i := uint32(0); i < epCount; i++ {
	//	var ctor xt.ConnectorI
	//	ctor, err = xt.NewTcpConnector(myNode.GetEndPoint(int(i)))
	//	c.Assert(err, IsNil)
	//	myCtors = append(myCtors, ctor)
	//}

	//cm := XXX RANDOMLY SELECTED MEMBER

	//// simplest test of Equal()
	//c.Assert(cm.Equal(cm), Equals, true)

	//// Serialize it
	//serialized := cm.String()

	//// Reverse the serialization
	//deserialized, rest, err := ParseClusterMember(serialized)
	//c.Assert(err, IsNil)
	//c.Assert(deserialized, Not(IsNil))
	//c.Assert(len(rest), Equals, 0)

	//// Verify that the deserialized ClusterMember is identical to the original
	//// First version:
	//c.Assert(deserialized.Equal(cm), Equals, true)

	//// Second version of identity test:
	//serialized2 := deserialized.String()
	//c.Assert(serialized2, Equals, serialized)
}
