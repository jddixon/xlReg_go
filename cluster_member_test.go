package reg

// xlReg_go/cluster_member_test.go

import (
	"crypto/rsa"
	"fmt"
	xh "github.com/jddixon/hamt_go"
	xr "github.com/jddixon/rnglib_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	. "gopkg.in/check.v1"
	"time"
)

// Create a test cluster using NewTestCluster() without using the 
// xlReg registry.
func (s *XLSuite) makeTestCluster(c *C, rng *xr.PRNG, name string, 
	nodeID *xi.NodeID, attrs uint64, size, epCount uint32) (tc *TestCluster) {

	var (
		members          []*ClusterMember
		ckPrivs, skPrivs []*rsa.PrivateKey
	)
	
	tc, err := NewTestCluster(name, nodeID, attrs, size, epCount)
	c.Assert(err, IsNil)
	c.Assert(tc, NotNil)
	// defer closing each member's acceptors, unless nil
	defer tc.CloseAcceptors()

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
			// DEBUG ------------------------------------------------
			acc := node.GetAcceptor(int(j))
			fmt.Printf("node %d, acceptor %d: %s\n", i, j, acc.String())
			// END --------------------------------------------------
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
		var (
			peer       *xn.Peer
			connectors []xt.ConnectorI
		)
		for j := uint32(0); j < epCount; j++ {
			var (
				ctor xt.ConnectorI
				ep   xt.EndPointI
			)
			ep = members[i].GetEndPoint(int(j))
			// DEBUG ------------------------------------------------
			fmt.Printf("node %d, endPoint %d: %s\n", i, j, ep.String())
			// END --------------------------------------------------
			ctor, err = xt.NewTcpConnector(ep)
			c.Assert(err, IsNil)
			c.Assert(ctor, NotNil)
			connectors = append(connectors, ctor)
		}
		peer, err = xn.NewPeer(members[i].GetName(), members[i].GetNodeID(),
			&ckPrivs[i].PublicKey, &skPrivs[i].PublicKey,
			nil, connectors) // overlays, connectors
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
	for i := uint32(0); i < size; i++ {
		member := members[i]
		if i == member.SelfIndex {
			// I can't be my own peer
			continue
		} else {
			mi := memberInfos[i]
			_, err = member.AddPeer(mi.Peer)
			c.Assert(err, IsNil)
		}
	}

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
	return
}
func (s *XLSuite) TestClusterMemberSerialization(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_CLUSTER_MEMBER_SERIALIZATION")
	}
	rng := xr.MakeSimpleRNG()

	// Generate a random test cluster
	name := rng.NextFileName(8)
	nodeID := s.makeANodeID(c, rng)
	attrs := uint64(rng.Int63())
	size := uint32(2 + rng.Intn(6))    // so from 2 to 7
	epCount := uint32(1 + rng.Intn(3)) // so from 1 to 3

	tc := s.makeTestCluster(c, rng, name, nodeID, attrs, size, epCount)

	// select a member randomly
	cmNdx := rng.Intn(int(size))
	cm := tc.ClMembers[cmNdx]

	// simplest test of Equal()
	c.Assert(cm.Equal(cm), Equals, true)

	// Serialize it
	serialized := cm.String()

	// close all acceptors (otherwise we get 'port in use' error)
	tc.CloseAcceptors()
	// allow time for things to settle; this might very rarely cause
	// us to lose a port
	time.Sleep(50 * time.Millisecond)

	// Reverse the serialization
	deserialized, rest, err := ParseClusterMember(serialized)
	c.Assert(err, IsNil)
	c.Assert(deserialized, Not(IsNil))
	c.Assert(len(rest), Equals, 0)

	// Verify that the deserialized ClusterMember is identical to the original
	// First version:
	c.Assert(deserialized.Equal(cm), Equals, true)

	// Second version of identity test:
	serialized2 := deserialized.String()
	c.Assert(serialized2, Equals, serialized)
}
