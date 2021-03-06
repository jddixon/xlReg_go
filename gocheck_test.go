package reg

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xcl "github.com/jddixon/xlCluster_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
	. "gopkg.in/check.v1"
	"strings"
	"testing"
)

// IF USING gocheck, need a file like this in each package=directory.

func Test(t *testing.T) { TestingT(t) }

type XLSuite struct{}

var _ = Suite(&XLSuite{})

const (
	VERBOSITY = 1
)

func (s *XLSuite) makeAnID(c *C, rng *xr.PRNG) (id []byte) {
	id = make([]byte, xu.SHA1_BIN_LEN)
	rng.NextBytes(id)
	return
}
func (s *XLSuite) makeANodeID(c *C, rng *xr.PRNG) (nodeID *xi.NodeID) {
	id := s.makeAnID(c, rng)
	nodeID, err := xi.New(id)
	c.Assert(err, IsNil)
	c.Assert(nodeID, Not(IsNil))
	return
}
func (s *XLSuite) makeAnRSAKey(c *C) (key *rsa.PrivateKey) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	c.Assert(err, IsNil)
	c.Assert(key, Not(IsNil))
	return key
}

// Creates a local (127.0.0.1) endPoint and adds it to the node.
// XXX This code was hacked from ../node/node_test.go.

func (s *XLSuite) makeALocalEndPoint(c *C, node *xn.Node) {
	addr := fmt.Sprintf("127.0.0.1:0")
	ep, err := xt.NewTcpEndPoint(addr)
	c.Assert(err, IsNil)
	c.Assert(ep, Not(IsNil))
	ndx, err := node.AddEndPoint(ep)
	c.Assert(err, IsNil)
	c.Assert(ndx, Equals, 0) // it's the only one
}

// Return an initialized and tested host, with a NodeID, commsKey,
// and sigKey.   XXX This code was hacked from ../node/node_test.go
// and then simplified a bit.

func (s *XLSuite) makeHostAndKeys(c *C, rng *xr.PRNG,
	namesInUse map[string]bool) (n *xn.Node, ckPriv, skPriv *rsa.PrivateKey) {

	var name string
	for {
		name = rng.NextFileName(8)
		for {
			first := string(name[0])
			if !strings.Contains(first, "0123456789") &&
				!strings.Contains(name, "-") {
				break
			}
			name = rng.NextFileName(8)
		}
		if _, ok := namesInUse[name]; !ok {
			// it's not in use
			namesInUse[name] = true
			break
		}
	}
	id := s.makeANodeID(c, rng)
	lfs := "tmp/" + hex.EncodeToString(id.Value())
	ckPriv = s.makeAnRSAKey(c)
	skPriv = s.makeAnRSAKey(c)

	n, err2 := xn.New(name, id, lfs, ckPriv, skPriv, nil, nil, nil)

	c.Assert(err2, IsNil)
	c.Assert(n, Not(IsNil))
	c.Assert(name, Equals, n.GetName())
	actualID := n.GetNodeID()
	c.Assert(true, Equals, id.Equal(actualID))
	// s.doKeyTests(c, n, rng)
	c.Assert(0, Equals, (*n).SizePeers())
	c.Assert(0, Equals, (*n).SizeOverlays())
	c.Assert(0, Equals, n.SizeConnections())
	c.Assert(lfs, Equals, n.GetLFS())
	return n, ckPriv, skPriv
}

// Make a client, a cluster member as seen by the registry server.
// Using functions must check to ensure members have unique names

func (s *XLSuite) makeAClientInfo(c *C, rng *xr.PRNG, epCount uint32) *ClientInfo {
	attrs := uint64(rng.Int63())
	var myEnds []string
	for i := uint32(0); i < epCount; i++ {
		myEnds = append(myEnds, "127.0.0.1:0")
	}
	bn, err := xn.NewBaseNode(
		rng.NextFileName(8),
		s.makeANodeID(c, rng),
		&s.makeAnRSAKey(c).PublicKey,
		&s.makeAnRSAKey(c).PublicKey,
		nil) // overlays
	c.Assert(err, IsNil)
	return &ClientInfo{
		Attrs:    attrs,
		MyEnds:   myEnds,
		BaseNode: *bn,
	}
}

// Make a cluster member as seen by registry clients.
// Using functions must check to ensure members have unique names

func (s *XLSuite) makeAMemberInfo(c *C, rng *xr.PRNG) *xcl.MemberInfo {
	attrs := uint64(rng.Int63())
	peer, err := xn.NewPeer(
		rng.NextFileName(8),
		s.makeANodeID(c, rng),
		&s.makeAnRSAKey(c).PublicKey,
		&s.makeAnRSAKey(c).PublicKey,
		nil, // overlays
		nil) // XXX CONNECTORS
	c.Assert(err, IsNil)
	return &xcl.MemberInfo{
		Attrs: attrs,
		Peer:  peer,
	}
} // GEEP

// Make a RegCluster for test purposes.  Cluster member names are guaranteed
// to be unique but the name of the cluster itself may not be.
//
// THIS IS THE REGISTRY'S VIEW OF A CLUSTER
func (s *XLSuite) makeARegCluster(c *C, rng *xr.PRNG, epCount, size uint32) (
	rc *RegCluster) {

	var err error
	c.Assert(MIN_CLUSTER_SIZE <= size && size <= MAX_CLUSTER_SIZE, Equals, true)

	attrs := uint64(rng.Int63())
	name := rng.NextFileName(8) // no guarantee of uniqueness
	id := s.makeANodeID(c, rng)

	rc, err = NewRegCluster(name, id, attrs, size, epCount)
	c.Assert(err, IsNil)

	for count := uint32(0); count < size; count++ {
		cm := s.makeAClientInfo(c, rng, epCount)
		for {
			if _, ok := rc.MembersByName[cm.GetName()]; ok {
				// name is in use, so try again
				cm = s.makeAClientInfo(c, rng, epCount)
			} else {
				err = rc.AddMember(cm)
				c.Assert(err, IsNil)
				break
			}
		}
	}
	return
}
