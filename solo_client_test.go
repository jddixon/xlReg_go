package reg

// xlReg_go/solo_client_test.go

import (
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xt "github.com/jddixon/xlTransport_go"
	xf "github.com/jddixon/xlUtil_go/lfs"
	. "gopkg.in/check.v1"
	"path"
)

func (s *XLSuite) TestSoloMember(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("TEST_SOLO_CLIENT")
	}

	rng := xr.MakeSimpleRNG()

	// 1.  create a new ephemeral server ----------------------------
	es, err := NewEphServer()
	c.Assert(err, IsNil)
	c.Assert(es, NotNil)

	server := es.Server

	c.Assert(&server.RegNode.ckPriv.PublicKey,
		DeepEquals, server.GetCommsPublicKey())
	serverName := server.GetName()
	serverID := server.GetNodeID()
	serverEnd := server.GetEndPoint(0)
	serverCK := server.GetCommsPublicKey()
	serverSK := server.GetSigPublicKey()
	c.Assert(serverEnd, NotNil)

	// start the mock server ------------------------------
	err = es.Run()
	c.Assert(err, IsNil)

	// 2. create the solo client ------------------------------------
	name := rng.NextFileName(8)
	lfs := path.Join("tmp", name)
	found, err := xf.PathExists(lfs)
	c.Assert(err, IsNil)
	for found {
		name = rng.NextFileName(8)
		lfs = path.Join("tmp", name)
		found, err = xf.PathExists(lfs)
		c.Assert(err, IsNil)
	}

	ep, err := xt.NewTcpEndPoint("127.0.0.1:0")
	c.Assert(err, IsNil)
	e := []xt.EndPointI{ep}

	sc, err := NewSoloMember(name, lfs, serverName, serverID, serverEnd,
		serverCK, serverSK, e)
	c.Assert(err, IsNil)
	c.Assert(sc, NotNil)

	// 3. run the client, which gets a nodeID from the server -------
	sc.Run()
	err = <-sc.DoneCh

	// 4.  verify that the client LFS exists and is correct ---------
	found, err = xf.PathExists(lfs)
	c.Assert(err, IsNil)
	c.Assert(found, Equals, true)

	// 5.  shut down the client -------------------------------------
	sc.Close() // should close any acceptors

	// 6.  stop the server by closing its acceptor ------------------
	es.Close()

}
