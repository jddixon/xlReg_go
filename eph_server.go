package reg

// xlReg_go/eph_server.go

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
)

var _ = fmt.Print

type EphServer struct {
	acc    xt.AcceptorI
	Server *RegServer
}

// An ephemeral server is primarily intended for use in testing.
// It does not persist registry information to disk.  It listens
// on a random port, 127.0.0.1:0.

func NewEphServer() (ms *EphServer, err error) {

	// Create an XLattice node with quasi-random parameters including
	// low-quality keys and an endPoint in 127.0.0.1, localhost.
	var (
		ckPriv, skPriv *rsa.PrivateKey
		rn             *RegNode
		ep             *xt.TcpEndPoint
		node           *xn.Node
		reg            *Registry
		server         *RegServer
	)

	rng := xr.MakeSimpleRNG()
	name := rng.NextFileName(16)
	idBuf := make([]byte, xu.SHA1_BIN_LEN)
	rng.NextBytes(idBuf)
	lfs := "tmp/" + hex.EncodeToString(idBuf)
	id, err := xi.New(nil)
	if err == nil {
		// XXX cheap keys, too weak for any serious use
		ckPriv, err = rsa.GenerateKey(rand.Reader, 1024)
		if err == nil {
			skPriv, err = rsa.GenerateKey(rand.Reader, 1024)
		}
	}
	if err == nil {
		ep, err = xt.NewTcpEndPoint("127.0.0.1:0")
		eps := []xt.EndPointI{ep}
		if err == nil {
			node, err = xn.New(name, id, lfs, ckPriv, skPriv, nil, eps, nil)
			if err == nil {
				err = node.OpenAcc() // so acceptors are now live
				if err == nil {
					rn, err = NewRegNode(node, ckPriv, skPriv)
					if err == nil {
						// DEBUG
						if rn == nil {
							fmt.Println("regNode is NIL!\n")
						} else {
							fmt.Printf("eph server listening on %s\n",
								rn.GetAcceptor(0).String())
						}
						// END
						// a registry with no clusters and no logger
						opt := &RegOptions{
							EndPoint:  ep, // not used
							Ephemeral: true,
							GlobalEndPoint: node.GetEndPoint(0),
							Lfs:       lfs, // redundant (is in node's BaseNode)
							Logger:    nil,
							K:         DEFAULT_K,
							M:         DEFAULT_M,
						}
						reg, err = NewRegistry(nil, rn, opt)
						if err == nil {
							server, err = NewRegServer(reg, true, 1)
							if err == nil {
								ms = &EphServer{
									acc:    rn.GetAcceptor(0),
									Server: server,
								}
							}
						}
					}
				}
			}
		}
	}
	return
}

// Start the ephemeral server running in a separate goroutine.

func (ms *EphServer) Start() (err error) {

	err = ms.Server.Start()
	return
}

func (ms *EphServer) GetAcceptor() (acc xt.AcceptorI) {
	return ms.acc
}

func (ms *EphServer) Stop() {
	ms.Server.Stop()
}
