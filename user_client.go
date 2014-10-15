package reg

// xlReg_go/user_client.go

import (
	"crypto/rsa"
	"fmt"
	xi "github.com/jddixon/xlNodeID_go"
	xt "github.com/jddixon/xlTransport_go"
)

var _ = fmt.Print

// The UserMember is created to enable the caller to join a cluster
// and learn information about the cluster's other members.  Once the
// client has learned that information, it is done.

// As implemented so far, this is an ephemeral client, meaning that it
// neither saves nor restores its Node; keys and such are generated for
// each instance.

// For practical use, it is essential that the UserMember create its
// Node when NewUserMember() is first called, but then save its
// configuration.  This is conventionally written to LFS/.xlattice/config.
// On subsequent the client reads its configuration file rather than
// regenerating keys, etc.

type UserMember struct {
	// members []MemberInfo		// XXX Nowhere used

	MemberMaker
}

func NewUserMember(
	name, lfs string, ckPriv, skPriv *rsa.PrivateKey,
	serverName string, serverID *xi.NodeID, serverEnd xt.EndPointI,
	serverCK, serverSK *rsa.PublicKey,
	clusterName string, clusterAttrs uint64, clusterID *xi.NodeID,
	size, epCount uint32, e []xt.EndPointI) (ac *UserMember, err error) {

	var attrs uint64

	if lfs == "" {
		attrs |= ATTR_EPHEMERAL
	}
	mn, err := NewMemberMaker(name, lfs, ckPriv, skPriv, attrs,
		serverName, serverID, serverEnd,
		serverCK, serverSK, //  *rsa.PublicKey,
		clusterName, clusterAttrs, clusterID, size,
		epCount, e)

	if err == nil {
		// Run() fills in clusterID
		ac = &UserMember{
			MemberMaker: *mn,
		}
	}
	return

}

// Start the member running in separate goroutine, so that this function
// is non-blocking.

func (uc *UserMember) Run() {

	mn := &uc.MemberMaker

	go func() {
		var (
			err      error
			version1 uint32
		)
		cnx, version2, err := mn.SessionSetup(version1)
		_ = version2 // not yet used
		if err == nil {
			err = mn.MemberAndOK()
		}
		if cnx != nil {
			defer cnx.Close()
		}
		// XXX MODIFY TO USE CLUSTER_ID PASSED TO UserMember
		// 2013-10-12 this is a join by cluster name
		if err == nil {
			err = mn.JoinAndReply()
		}
		if err == nil {
			err = mn.GetAndMembers()
		}
		// DEBUG
		var nilMembers []int
		for i := 0; i < len(uc.Members); i++ {
			if uc.Members[i] == nil {
				nilMembers = append(nilMembers, i)
			}
		}
		if len(nilMembers) > 0 {
			fmt.Printf("UserMember.Run() after Get finds nil members: %v\n",
				nilMembers)
		}
		// END
		if err == nil {
			err = mn.ByeAndAck()
		}
		mn.DoneCh <- err
	}()
}
