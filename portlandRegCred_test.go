package reg

// xlReg_go/portlandRegCred_test.go

import (
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	//xu "github.com/jddixon/xlUtil_go"
	xf "github.com/jddixon/xlUtil_go/lfs"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"path"
)

func (s *XLSuite) TestPortlandRegCred(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_PORTLAND_REG_CRED")
	}
	rng := xr.MakeSimpleRNG()
	_ = rng

	// read our local copy of the reg cred
	rcData, err := ioutil.ReadFile("portlandRegCred.dat")
	c.Assert(err, IsNil)
	rc, err := ParseRegCred(string(rcData))
	c.Assert(err, IsNil)
	c.Assert(rc, NotNil)

	// DEBUG
	fmt.Println("portlandRegCred_test PORTLAND - A PUZZLE")
	// END

	// set up the client --------------------------------------------
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
	c.Assert(e, NotNil)

	nodeID, err := xi.New(nil)
	c.Assert(err, IsNil)

	node, err := xn.NewNew(name, nodeID, lfs)
	c.Assert(err, IsNil)

	// DEBUG
	fmt.Printf("Portland client is at %v; lfs is %s\n",
		e, lfs)
	// END

	// set up its relationship to the server ------------------------
	serverName := rc.Name
	serverID := rc.ID
	serverEnd := rc.EndPoints
	c.Assert(serverEnd, NotNil)
	c.Assert(len(serverEnd) > 0, Equals, true)
	c.Assert(serverEnd[0], NotNil)
	// XXX TOO RIGID
	c.Assert(serverEnd[0].String(), Equals, "TcpEndPoint: 54.186.197.123:56789")
	// END
	serverCK := rc.CommsPubKey
	serverSK := rc.SigPubKey

	sc, err := NewSoloMember(node, serverName, serverID, serverEnd[0],
		serverCK, serverSK, e)
	c.Assert(err, IsNil)
	c.Assert(sc, NotNil)

	// DEBUG
	fmt.Println("SoloMember CREATED")

	// END

	// 3. run the client
	sc.Start()
	err = <-sc.DoneCh

	// 4.  verify that the client LFS exists and is correct ---------
	found, err = xf.PathExists(lfs)
	c.Assert(err, IsNil)
	c.Assert(found, Equals, true)

	// 5.  shut down the client -------------------------------------
	sc.CloseAcc() // should close any acceptors

}
