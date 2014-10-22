package reg

// xlReg_go/admin_client.go

import (
	"crypto/rsa"
	"fmt"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
)

var _ = fmt.Print

// In the current implementation, AdminMember's purpose is to register
// clusters, so it sets up a session with the registry, (calls SessionSetup),
// identifies itself with a Client/ClientOK sequence, uses Create/CreateReply
// to register the cluster, and then ends with Bye/Ack.  It then returns
// the clusterID and size to the caller.

// As implemented so far, this is an ephemeral client, meaning that it
// neither saves nor restores its Node; keys and such are generated for
// each instance.

type AdminMember struct {
	// In this implementation, AdminMember is a one-shot, launched
	// to create a single cluster

	MemberMaker
}

func NewAdminMember(
	serverName string, serverID *xi.NodeID, serverEnd xt.EndPointI,
	serverCK, serverSK *rsa.PublicKey,
	clusterName string, clusterAttrs uint64,
	size, epCount uint32, e []xt.EndPointI) (
	ac *AdminMember, err error) {

	nodeID, err := xi.New(nil)
	// DEBUG
	fmt.Printf("  admin ID %x\n", nodeID.Value())
	// END
	if err == nil {
		node, err := xn.NewNew("admin", nodeID, "") // name, id, lfs
		if err == nil {

			cn, err := NewMemberMaker(node,
				ATTR_ADMIN|ATTR_SOLO|ATTR_EPHEMERAL,
				serverName, serverID, serverEnd, serverCK, serverSK,
				clusterName, clusterAttrs, nil, size, epCount, e)

			if err == nil {
				// Run() fills in clusterID
				ac = &AdminMember{
					MemberMaker: *cn,
				}
			}
		}
	}
	return
}

// Start the client running in separate goroutine, so that this function
// is non-blocking.

func (ac *AdminMember) Run() {

	mm := &ac.MemberMaker

	go func() {
		var (
			version1 uint32
		)
		cnx, version2, err := mm.SessionSetup(version1)
		_ = version2 // not yet used
		// DEBUG
		fmt.Printf("  after SessionSetup err is %v\n", err)
		// END
		if err == nil {
			err = mm.MemberAndOK()
			// DEBUG
			fmt.Printf("  after MemberAndOK err is %v\n", err)
			// END
			if err == nil {
				err = mm.CreateAndReply()
				// DEBUG
				fmt.Printf("  after CreateAndReply err is %v\n", err)
				// END
				if err == nil {
					err = mm.ByeAndAck()
				}
			}
		}
		// END OF RUN ===============================================
		if cnx != nil {
			cnx.Close()
		}
		// DEBUG
		fmt.Printf("AdminMember.Run() : exiting; err %v\n", err)
		fmt.Printf("  ClusterID %x\n", mm.ClusterID.Value())
		// END
		mm.DoneCh <- err
	}()
}
