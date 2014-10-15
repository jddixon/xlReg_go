package reg

// xlReg_go/solo_client.go

import (
	"crypto/rsa"
	"fmt"
	xi "github.com/jddixon/xlNodeID_go"
	xt "github.com/jddixon/xlTransport_go"
)

var _ = fmt.Print

// In the current implementation, a SoloMember provides a simple way to
// create an XLattice Node with a unique NodeID, private RSA keys,
// and an initialized LFS with the configuration stored in
// LFS/.xlattice/node.config with a default mode of 0400.
//
// It requires a registry to give it its NodeID.
//
// In other words, use a SoloMember to create and persist an XLattice
// node that will NOT be a member of a cluster but WILL have its
// configuration saved to permanent storage, to its local file system
// (LFS).

type SoloMember struct {
	// In this implementation, SoloMember is a one-shot, launched
	// to create a solitary node.

	MemberMaker
}

func NewSoloMember(name, lfs string,
	serverName string, serverID *xi.NodeID, serverEnd xt.EndPointI,
	serverCK, serverSK *rsa.PublicKey,
	e []xt.EndPointI) (
	sc *SoloMember, err error) {

	cn, err := NewMemberMaker(name, lfs, nil, nil, ATTR_SOLO,
		serverName, serverID, serverEnd, serverCK, serverSK,
		"", uint64(0), nil, 0, // no cluster
		uint32(len(e)), e)

	if err == nil {
		// Run() fills in clusterID
		sc = &SoloMember{
			MemberMaker: *cn,
		}
	}
	return
}

// Start the client running in separate goroutine, so that this function
// is non-blocking.

func (sc *SoloMember) Run() {

	cn := &sc.MemberMaker

	go func() {
		var (
			version1 uint32
		)
		cnx, version2, err := cn.SessionSetup(version1)
		_ = version2 // not yet used
		if err == nil {
			err = cn.MemberAndOK()
		}
		if err == nil {
			err = cn.ByeAndAck()
		}
		// END OF RUN ===============================================

		if cnx != nil {
			cnx.Close()
		}
		// Create the Node and write its configuration to the usual place
		// in the file system: LFS/.xlattice/node.config.
		err = cn.PersistNode()
		cn.DoneCh <- err
	}()
}
