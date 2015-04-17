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

// In the current implementation, AdminClient's purpose is to register
// clusters, so it sets up a session with the registry, (calls SessionSetup),
// identifies itself with a Client/ClientOK sequence, uses Create/CreateReply
// to register the cluster, and then ends with Bye/Ack.  It then returns
// the clusterID and size to the caller.

// As implemented so far, this is an ephemeral client, meaning that it
// neither saves nor restores its Node; keys and such are generated for
// each instance.

type AdminClient struct {
	// In this implementation, AdminClient is a one-shot, launched
	// to create a single cluster

	MemberMaker
}

func NewAdminClient(
	serverName string, serverID *xi.NodeID, serverEnd xt.EndPointI,
	serverCK, serverSK *rsa.PublicKey,
	clusterName string, clusterAttrs uint64,
	size, epCount uint32, e []xt.EndPointI) (
	ac *AdminClient, err error) {

	nodeID, err := xi.New(nil)
	// DEBUG
	fmt.Printf("NewAdminClient: admin ID %x\n", nodeID.Value())
	// END
	if err == nil {
		node, err := xn.NewNew("admin", nodeID, "") // name, id, lfs
		if err == nil {
			cn, err := NewMemberMaker(node,
				ATTR_ADMIN|ATTR_SOLO|ATTR_EPHEMERAL,
				serverName, serverID, serverEnd, serverCK, serverSK,
				clusterName, clusterAttrs, nil, size, epCount, e)

			if err == nil {
				// Start() fills in clusterID
				ac = &AdminClient{
					MemberMaker: *cn,
				}
				// we do NOT invoke node.OpenAcc() on adminClients
				// DEBUG
				fmt.Println("   NewAdminClient: OK exit")
				// END
			}
		}
	}
	return
}

// Start the client running in separate goroutine, so that this function
// is non-blocking.
//
// XXX If the xlReg server is not running, this gets an error:
//   dial tcp 127.0.0.1:0: connection refused
// and then a panic because the nodeID is nil

func (ac *AdminClient) Start() {

	// DEBUG
	fmt.Printf("AdminClient.Start\n")
	// END

	mm := &ac.MemberMaker

	go func() {
		var (
			version1 uint32
		)
		cnx, version2, err := mm.SessionSetup(version1)
		_ = version2 // not yet used
		// DEBUG
		fmt.Printf("  AC.S: after SessionSetup err is %v\n", err)
		// END
		if err == nil {
			err = mm.MemberAndOK()
			// DEBUG
			fmt.Printf("  AC.S: after MemberAndOK err is %v\n", err)
			// END
			if err == nil {
				err = mm.CreateAndReply()
				// DEBUG
				fmt.Printf("  AC.S: after CreateAndReply err is %v\n", err)
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
		fmt.Printf("    AC.S: exiting; err %v\n", err)
		if mm.ClusterID == nil {
			fmt.Printf("    NIL mm.ClusterID\n")
		} else {
			fmt.Printf("  ClusterID %x\n", mm.ClusterID.Value())
		}
		// END
		mm.DoneCh <- err
	}()
}
