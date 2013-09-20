package reg

// xlattice_go/reg/mock_server.go

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	//xc "github.com/jddixon/xlattice_go/crypto"
	//xn "github.com/jddixon/xlattice_go/node"
	xi "github.com/jddixon/xlattice_go/nodeID"
	xr "github.com/jddixon/xlattice_go/rnglib"
	xt "github.com/jddixon/xlattice_go/transport"
)

var _ = fmt.Print

type MockServer struct {
	acc         xt.AcceptorI // a convenience
	clusterName string
	clusterID   *xi.NodeID
	size        int
	RegNode
}

// A Mock Server is primarily intended for use in testing.  It contains
// a registry which handles one and only one cluster of a fixed size.

func NewMockServer(clusterName string, clusterID *xi.NodeID, size int) (
	ms *MockServer, err error) {

	if clusterName == "" || clusterID == nil {
		err = MissingClusterNameOrID
	} else if size < 2 {
		err = ClusterMustHaveTwo
	}
	if err != nil {
		return
	}

	// Create an XLattice node with quasi-random parameters including
	// low-quality keys and an endPoint in 127.0.0.1, localhost.

	var ckPriv, skPriv *rsa.PrivateKey
	var rn *RegNode
	var ep *xt.TcpEndPoint

	rng := xr.MakeSimpleRNG()
	name := rng.NextFileName(16)
	idBuf := make([]byte, SHA1_LEN)
	rng.NextBytes(&idBuf)
	lfs := "tmp/" + hex.EncodeToString(idBuf)
	id, err := xi.New(idBuf)
	if err == nil {
		// XXX cheap keys, not meant for any serious use
		ckPriv, err = rsa.GenerateKey(rand.Reader, 512)
		if err == nil {
			skPriv, err = rsa.GenerateKey(rand.Reader, 512)
		}
	}
	if err == nil {
		ep, err = xt.NewTcpEndPoint("127.0.0.1:0")
	}
	if err == nil {
		rn, err = NewRegNode(name, id, lfs, ckPriv, skPriv, nil, ep)
	}
	if err == nil {
		ms = &MockServer{
			acc:         rn.GetAcceptor(0),
			clusterName: clusterName,
			clusterID:   clusterID,
			size:        size,
			RegNode:     *rn,
		}
	}
	return
}

func (ms *MockServer) ClientHandler(cnx xt.ConnectionI) (err error) {
	// hello and reply set up the AES iv and key

	// Expect ClientMsg

	// Answer with ClientOK or error

	// Expect CreateMsg

	// Answer with CreateReply

	// Expect JoinMsg

	// Answer with JoinReply

	// Expect Get

	// Answer with Members

	// Repeat Get/Members or Expect Bye

	// Send Ack and close connection

	return
}

// Start the mock server running in a separate goroutine.  As each
// client connects its connection is passed to a  handler running in
// a separate goroutine.

func (ms *MockServer) Run() (err error) {

	// XXX STUB XXX

	return
}

func (ms *MockServer) Close() {
	if ms.acc != nil {
		ms.acc.Close()
	}
}
