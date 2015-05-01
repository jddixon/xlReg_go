package reg

// xlReg_go/solo_client.go

import (
	"crypto/rsa"
	"fmt"
	xcl "github.com/jddixon/xlCluster_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	"os"
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

func NewSoloMember(node *xn.Node,
	serverName string, serverID *xi.NodeID, serverEnd xt.EndPointI,
	serverCK, serverSK *rsa.PublicKey,
	e []xt.EndPointI) (
	sc *SoloMember, err error) {

	cn, err := NewMemberMaker(node, xcl.ATTR_SOLO,
		serverName, serverID, serverEnd, serverCK, serverSK,
		"", uint64(0), nil, 0, // no cluster
		uint32(len(e)), e)

	if err == nil {
		// Start() fills in clusterID
		sc = &SoloMember{
			MemberMaker: *cn,
		}
	}
	return
}

// Start the client running in separate goroutine, so that this function
// is non-blocking.

func (sc *SoloMember) Start() {

	cn := &sc.MemberMaker

	go func() {
		// DEBUG
		fo, err := os.Create("junk.soloClient.log")
		if err != nil {
			panic(err)
		}
		defer fo.Close()
		// END
		var (
			version1 uint32
		)
		cnx, version2, err := cn.SessionSetup(version1)
		_ = version2 // not yet used
		defer func() {
			if cnx != nil {
				cnx.Close()
			}
		}()
		// DEBUG
		if err != nil {
			fo.WriteString(fmt.Sprintf("err fromSessionSetup is %s\n",
				err.Error()))
		}
		fo.WriteString(fmt.Sprintf("lfs is %s\n", cn.GetLFS())) // DEBUG
		// END

		if err == nil {
			err = cn.MemberAndOK()
			// DEBUG
			fo.WriteString("MemberAndOK done\n")
			if err != nil {
				fo.WriteString(fmt.Sprintf("  err was %s\n", err))
			}
			// END
			if err == nil {
				err = cn.ByeAndAck()
				// DEBUG
				fo.WriteString("ByeAndAck done\n")
				if err != nil {
					fo.WriteString(fmt.Sprintf("  err was %s\n", err))
				}
				// END
				// END OF RUN =======================================
				if err == nil {
					// WriteString configuration to // the usual place in the
					// file system:   LFS/.xlattice/node.config.
					err = cn.PersistNode()
					// DEBUG
					fo.WriteString("Node persisted\n")
					if err != nil {
						fo.WriteString(fmt.Sprintf("  err was %s\n", err))
					}
					// END
				}
			}
		}
		fo.Sync() // DEBUG
		cn.DoneCh <- err
	}()
}
