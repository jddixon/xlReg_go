package reg

// xlReg_go/msg_handlers.go

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex" // DEBUG
	e "errors"
	"fmt"
	ha "github.com/jddixon/hamt_go"
	xc "github.com/jddixon/xlCrypto_go"
	xi "github.com/jddixon/xlNodeID_go"
	xu "github.com/jddixon/xlUtil_go"
)

var _ = fmt.Print

// XXX Possibly a problem, possibly not: the message number / sequence
// number has disappeared.

/////////////////////////////////////////////////////////////////////
// AES-BASED MESSAGE PAIRS
// All of these functions have the same signature, so that they can
// be invoked through a dispatch table.
/////////////////////////////////////////////////////////////////////

// Dispatch table entry where a client message received is inappropriate
// the the state of the connection.  For example, if we haven't yet
// received information about the client, we should not be receiving a
// Join or Get message.
func badCombo(h *InHandler) {
	h.errOut = RcvdInvalidMsgForState
}

// CLIENT AND CLIENT_OK =============================================

// Handle the message which gives us information about the client and
// so associates this connection with a specific user.

func doClientMsg(h *InHandler) {
	var err error
	defer func() {
		h.errOut = err
	}()

	// DEBUG
	regName := h.reg.GetName()
	h.reg.Logger.Printf("doClientMsg: regName is %s\n", regName)
	// END

	// Examine incoming message -------------------------------------
	var (
		name   string
		attrs  uint64
		nodeID *xi.NodeID
		ck, sk *rsa.PublicKey
		myEnds []string
		hash   []byte
		cm     *ClientInfo
	)

	// XXX We should accept EITHER clientName + token OR clientID
	// This implementation only accepts a token.

	clientMsg := h.msgIn
	clientSpecs := clientMsg.GetMemberSpecs()
	name = clientSpecs.GetName()
	attrs = clientSpecs.GetAttrs()
	ckBytes := clientSpecs.GetCommsKey()
	skBytes := clientSpecs.GetSigKey()
	digSig := clientSpecs.GetDigSig()

	ck, err = xc.RSAPubKeyFromWire(ckBytes)
	if err == nil {
		sk, err = xc.RSAPubKeyFromWire(skBytes)
		if err == nil {
			myEnds = clientSpecs.GetMyEnds() // a string array
		}
	}
	if err == nil {
		// calculate hash over fields in canonical order XXX EXCLUDING ID
		aBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(aBytes, attrs)
		d := sha1.New()
		d.Write([]byte(name))
		d.Write(aBytes)
		d.Write(ckBytes)
		d.Write(skBytes)
		for i := 0; i < len(myEnds); i++ {
			d.Write([]byte(myEnds[i]))
		}
		hash = d.Sum(nil)
		// verify the digital signature
		err = rsa.VerifyPKCS1v15(sk, crypto.SHA1, hash, digSig)
	}
	if err == nil {
		id := clientSpecs.GetID()
		// DEBUG
		if id == nil {
			h.reg.Logger.Println("  doClientMsg: id from Specs is NIL")
		} else {
			h.reg.Logger.Printf("  doClientMsg: id from Specs is %x\n", id)
		}
		// END
		if id == nil {
			nodeID, err = h.reg.InsertUniqueNodeID()
			// DEBUG
			h.reg.Logger.Printf("  doClientMsg: inserting %x returned %v\n", id, err)
			// END
			if err == nil {
				id := nodeID.Value()
				// this is echoed to the console
				h.reg.Logger.Printf("doClientMsg: assigning new MemberID %xi, user %s",
					id, name)
			}
		} else {
			// must be known to the registry
			nodeID, err = xi.New(id)
			if err == nil {
				var found bool
				found, err = h.reg.ContainsID(nodeID)
				if err == nil && !found {
					err = h.reg.InsertID(nodeID)
				}
			}
		}
	}
	// Take appropriate action --------------------------------------
	if err == nil || err == IDAlreadyInUse {
		// The appropriate action is to hang a token for this client off
		// the InHandler.
		cm, err = NewClientInfo(name, nodeID, ck, sk, attrs, myEnds)
		if err == nil {
			h.thisClient = cm
		}
	}
	if err == nil {
		// Prepare reply to client --------------------------------------
		// In this implementation We simply accept the client's proposed
		// attrs and ID.
		op := XLRegMsg_MemberOK
		h.msgOut = &XLRegMsg{
			Op:          &op,
			MemberID:    nodeID.Value(),
			MemberAttrs: &attrs, // in production, review and limit
		}

		// Set exit state -----------------------------------------------
		h.exitState = CLIENT_DETAILS_RCVD
	}
}

// CREATE AND CREATE_REPLY ==========================================

// Handle the Create message which associates a unique name with a
// cluster and specifies its proposed size.  The server replies with the
// cluster ID and its server-assigned size.
//
// XXX This implementation does not handle cluster attrs.

func doCreateMsg(h *InHandler) {
	var err error
	defer func() {
		h.errOut = err
	}()
	// Examine incoming message -------------------------------------
	var (
		clusterID *xi.NodeID
		index     int
	)

	createMsg := h.msgIn
	clusterName := createMsg.GetClusterName()
	clusterAttrs := createMsg.GetClusterAttrs()
	clusterMaxSize := createMsg.GetClusterMaxSize()
	endPointCount := createMsg.GetEndPointCount()

	// Take appropriate action --------------------------------------

	// Determine whether the cluster exists.  If it does, we will just
	// use its existing properties.

	h.reg.mu.RLock()
	cluster, exists := h.reg.ClustersByName[clusterName]
	h.reg.mu.RUnlock()

	if exists {
		h.reg.Logger.Printf("doCreateMsg: cluster %s already exists\n", clusterName)
		clusterMaxSize = cluster.maxSize
		clusterAttrs = cluster.Attrs
		endPointCount = cluster.epCount
		if cluster.ID == nil {
			h.reg.Logger.Println("  no ID for cluster %s\n", clusterName)
			clusterID, _ = xi.New(nil)
		} else {
			clusterID, _ = xi.New(cluster.ID)
		}
		// XXX index not assigned
	} else {
		h.reg.Logger.Printf("doCreateMsg: new cluster %s\n", clusterName)
		attrs := uint64(0)
		if clusterMaxSize < MIN_CLUSTER_SIZE {
			clusterMaxSize = MIN_CLUSTER_SIZE
		} else if clusterMaxSize > MAX_CLUSTER_SIZE {
			clusterMaxSize = MAX_CLUSTER_SIZE
		}
		// Assign a quasi-random cluster ID, adding it to the registry
		clusterID, err = h.reg.InsertUniqueNodeID()
		if err == nil {
			cluster, err = NewRegCluster(clusterName, clusterID, attrs,
				clusterMaxSize, endPointCount)
			h.reg.Logger.Printf("cluster %s assigning ID %x\n",
				clusterName, clusterID.Value())
		}
		if err == nil {
			h.cluster = cluster
			index, err = h.reg.AddCluster(cluster)
			// XXX index not used
		}
	}
	_ = index // INDEX IS NOT BEING USED

	if err == nil {
		// Prepare reply to client --------------------------------------
		op := XLRegMsg_CreateReply
		id := clusterID.Value()
		h.msgOut = &XLRegMsg{
			Op:             &op,
			ClusterID:      id,
			ClusterMaxSize: &clusterMaxSize,
			ClusterAttrs:   &clusterAttrs,
			EndPointCount:  &endPointCount,
		}
		// Set exit state -----------------------------------------------
		h.exitState = CREATE_REQUEST_RCVD
	}
}

// JOIN AND JOIN_REPLY ==============================================

// Tie this session to a specific cluster, either by supplying its
// name or using the clusterID.  Return the cluster ID and its size.
//

func doJoinMsg(h *InHandler) {

	var err error
	defer func() {
		h.errOut = err
	}()
	// Examine incoming message -------------------------------------
	var (
		cluster        *RegCluster
		clusterName    string
		clusterID      []byte
		clusterMaxSize uint32
		endPointCount  uint32
	)
	joinMsg := h.msgIn

	// Take appropriate action --------------------------------------

	// Accept either cluster name or id.  If it's just the name,
	// attempt to retrieve the ID; it's an error if it does not exist
	// in the registry.  . In either case use the ID to retrieve the size.

	clusterName = joinMsg.GetClusterName() // will be "" if absent
	clusterID = joinMsg.GetClusterID()     // will be nil if absent

	if clusterID != nil {
		h.reg.Logger.Printf("JOIN: cluster %x, new member %s\n",
			clusterID, h.thisClient.GetName())
	} else {
		h.reg.Logger.Printf("JOIN: cluster %s, new member %s\n",
			clusterName, h.thisClient.GetName())
	}

	if clusterID == nil && clusterName == "" {
		// if neither is present, we will use any cluster already
		// associated with this connection
		if h.cluster != nil {
			cluster = h.cluster
		} else {
			err = MissingClusterNameOrID
		}
	} else if clusterID != nil {
		var kluster interface{}

		// convert the clusterID into a HAMT BytesKey
		bKey, err := ha.NewBytesKey(clusterID)
		if err == nil {
			// if an ID has Leen defined, we will try to use that
			h.reg.mu.RLock()
			kluster, err = h.reg.ClustersByID.Find(bKey)
			h.reg.mu.RUnlock()
			if kluster == nil {
				msg := fmt.Sprintf("can't find cluster with ID %s",
					hex.EncodeToString(clusterID))
				// DEBUG
				h.reg.Logger.Printf("%s\n", msg)
				// END
				err = e.New(msg)
			} else {
				cluster = kluster.(*RegCluster)
			}
		}
	} else {
		// we have no ID and clusterName is not nil, so we will try to use that
		var ok bool
		h.reg.mu.RLock()
		if cluster, ok = h.reg.ClustersByName[clusterName]; !ok {
			err = CantFindClusterByName
		}
		h.reg.mu.RUnlock()
	}
	if err == nil {
		// if we get here, cluster is not nil
		err = cluster.AddMember(h.thisClient)
	}
	if err == nil {
		h.reg.Logger.Printf("cluster %x, new member %s\n",
			cluster.ID, h.thisClient.GetName())

		// Prepare reply to client ----------------------------------
		h.cluster = cluster
		clusterID = cluster.ID
		clusterAttrs := cluster.Attrs
		clusterMaxSize = h.cluster.maxSize
		endPointCount = h.cluster.epCount

		op := XLRegMsg_JoinReply
		h.msgOut = &XLRegMsg{
			Op:             &op,
			ClusterID:      clusterID,
			ClusterAttrs:   &clusterAttrs,
			ClusterMaxSize: &clusterMaxSize,
			EndPointCount:  &endPointCount,
		}
		// Set exit state -------------------------------------------
		h.exitState = JOIN_RCVD
	}
	if err != nil {
		h.reg.Logger.Printf("cluster %x, new member %s, ERROR %s\n",
			cluster.ID, h.thisClient.GetName(), err.Error())
	}
}

// GET AND MEMBERS ==================================================

// Fetch from the registry details for the specified members for the
// cluster.  The cluster is identified by its ID.  Members requested
// are specified using a bit vector: we assume that members are stored
// in the order in which they joined, so if the Nth bit is set, we
// want a copy of the details for that member.  It is an error if the
// clusterID does not correspond to an existing cluster.  It is not
// an error if a member cannot be found for one of the bits set: the
// server returns a bit vector specifying which member tokens are being
// returned.

func doGetMsg(h *InHandler) {
	var (
		cluster *RegCluster
		err     error
	)
	defer func() {
		h.errOut = err
	}()
	// Examine incoming message -------------------------------------
	getMsg := h.msgIn
	clusterID := getMsg.GetClusterID()
	whichRequested := xu.NewBitMap64(getMsg.GetWhich())

	// Take appropriate action --------------------------------------
	var tokens []*XLRegMsg_Token
	whichReturned := xu.NewBitMap64(0)

	// Put the type assertion within the critical section because on
	// 2013-12-04 I observed a panic with the error message saying
	// that the assertion failed because kluster was nil, which should
	// be impossible.

	// convert clusterID into a BytesKey
	bKey, err := ha.NewBytesKey(clusterID)
	if err == nil {
		var kluster interface{}
		h.reg.mu.RLock() // <-- LOCK --------------------------
		kluster, err = h.reg.ClustersByID.Find(bKey)
		if kluster == nil {
			msg := fmt.Sprintf("doGetMsg: can't find cluster with ID %s",
				hex.EncodeToString(clusterID))
			// DEBUG
			h.reg.Logger.Printf("%s\n", msg)
			// END
			err = e.New(msg)
		} else {
			cluster = kluster.(*RegCluster)
		}
		h.reg.mu.RUnlock() // <-- UNLOCK ----------------------
	}

	if err == nil {
		size := cluster.Size()       // actual size, not MaxSize
		if size > MAX_CLUSTER_SIZE { // yes, should be impossible
			size = MAX_CLUSTER_SIZE
		}
		// XXX UNDESIRABLE CAST
		weHave := xu.LowNMap(uint(size))
		whichToSend := whichRequested.Intersection(weHave)
		for i := uint32(0); i < size; i++ {
			// XXX UNDESIRABLE CAST
			if whichToSend.Test(uint(i)) { // they want this one
				member := cluster.Members[i]
				token, err := member.Token()
				if err == nil {
					tokens = append(tokens, token)
					// XXX UNDESIRABLE CAST
					whichReturned = whichReturned.Set(uint(i))
				} else {
					break
				}
			}
		}
	}
	if err == nil {
		// Prepare reply to client --------------------------------------
		op := XLRegMsg_ClusterMembers
		h.msgOut = &XLRegMsg{
			Op:        &op,
			ClusterID: clusterID,
			Which:     &whichReturned.Bits,
			Tokens:    tokens,
		}
		// Set exit state -----------------------------------------------
		h.exitState = JOIN_RCVD // the JOIN is intentional !
	}
}

// BYE AND ACK ======================================================

func doByeMsg(h *InHandler) {
	var err error
	defer func() {
		h.errOut = err
	}()

	// Examine incoming message -------------------------------------
	//ByeMsg := h.msgIn

	// Take appropriate action --------------------------------------
	// nothing to do

	// Prepare reply to client --------------------------------------
	op := XLRegMsg_Ack
	h.msgOut = &XLRegMsg{
		Op: &op,
	}
	// Set exit state -----------------------------------------------
	h.exitState = BYE_RCVD
}
