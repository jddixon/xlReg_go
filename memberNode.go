package reg

// xlReg_go/memberNode.go

import (
	"bytes"
	"crypto"
	//"crypto/aes"
	//"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xcl "github.com/jddixon/xlCluster_go"
	xc "github.com/jddixon/xlCrypto_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xa "github.com/jddixon/xlProtocol_go/aes_cnx"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
	xf "github.com/jddixon/xlUtil_go/lfs"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

var _ = fmt.Print // DEBUG

// member states
const (
	MEMBER_START = iota
	HELLO_SENT
	MEMBER_SENT
	CLUSTER_SENT
	JOIN_SENT
	GET_SENT
	BYE_SENT
	MEMBER_CLOSED
)

type MemberMaker struct {

	// EPHEMERAL INFORMATION: should not be serialized ==============
	DoneCh          chan error
	ProposedAttrs   uint64
	ProposedVersion uint32 // proposed by member

	// serverVersion xu.DecimalVersion		// missing

	regProtoVersion uint32 // protocol version used to talk to registry

	CnxHandler
	RegPeer *xn.Peer
	xcl.ClusterMember
}

// Returns a copy of the node's comms private RSA key
func (mm *MemberMaker) GetCKPriv() rsa.PrivateKey {
	return *mm.GetCommsPrivateKey()
}

// Returns a copy of the node's sig private RSA key
func (mm *MemberMaker) GetSKPriv() rsa.PrivateKey {
	return *mm.GetSigPrivateKey()
}

// Create just the Node for this member and write it to the conventional
// place in the file system.
func (mm *MemberMaker) PersistNode() (err error) {

	var (
		config string
	)

	// XXX check attrs, etc

	lfs := mm.ClusterMember.Node.GetLFS()
	pathToCfgDir := path.Join(lfs, ".xlattice")
	pathToCfgFile := path.Join(pathToCfgDir, "node.config")
	found, err := xf.PathExists(pathToCfgDir)
	if err == nil && !found {
		err = xf.CheckLFS(pathToCfgDir, 0750)
	}
	if err == nil {
		//mm.Node = *node
		config = mm.Node.String()
	}
	if err == nil {
		err = ioutil.WriteFile(pathToCfgFile, []byte(config), 0600)
	}
	return
}

// Create the Node for this client and write the serialized ClusterMember
// to the conventional place in the file system.
func (mn *MemberMaker) PersistClusterMember() (err error) {

	var config string

	// XXX check attrs, etc

	lfs := mn.ClusterMember.Node.GetLFS()
	pathToCfgDir := path.Join(lfs, ".xlattice")
	pathToCfgFile := path.Join(pathToCfgDir, "cluster.member.config")

	_, err = os.Stat(pathToCfgDir)
	if os.IsNotExist(err) {
		err = xf.CheckLFS(pathToCfgDir, 0740)
	}

	if err == nil {
		config = mn.ClusterMember.String()
		if err == nil {
			err = ioutil.WriteFile(pathToCfgFile, []byte(config), 0600)
		}
	}
	return
}

// Given contact information for a registry and the name of a cluster,
// the client joins the cluster, collects information on the other members,
// and terminates when it has info on the entire membership.
func NewMemberMaker(
	node *xn.Node, attrs uint64,
	regName string, regID *xi.NodeID, regEnd xt.EndPointI,
	regCK, regSK *rsa.PublicKey,
	clusterName string, clusterAttrs uint64, clusterID *xi.NodeID,
	size, epCount uint32, endPoints []xt.EndPointI) (
	mm *MemberMaker, err error) {

	var (
		cm      *xcl.ClusterMember
		isAdmin = (attrs & xcl.ATTR_ADMIN) != 0
		regPeer *xn.Peer
	)
	// sanity checks on parameter list
	if node == nil {
		err = MissingNode
	} else {
		if regName == "" || regID == nil || regEnd == nil ||
			regCK == nil {

			err = MissingServerInfo
		}
		if err == nil {
			// DEBUG
			fmt.Printf("NemMemberMaker: regEnd is %s\n", regEnd.String())
			// END
			if (attrs & xcl.ATTR_SOLO) == uint64(0) {
				if clusterName == "" {
					err = MissingClusterNameOrID
					if err == nil && size < uint32(1) {
						// err = ClusterMustHaveTwo
						err = ClusterMustHaveMember
					}
				}
				if err == nil {
					// if the client is an admin client epCount applies
					// to the cluster
					if epCount < uint32(1) {
						epCount = uint32(1)
					}
					if !isAdmin {
						// XXX There is some confusion here: we don't require
						// that all members have the same number of endpoints
						actualEPCount := uint32(len(endPoints))
						if actualEPCount == 0 {
							err = MemberMustHaveEndPoint
						} else if epCount > actualEPCount {
							epCount = actualEPCount
						}
						for i := 0; i < int(epCount); i++ {
							_, err = node.AddEndPoint(endPoints[i])
						}
					}
				}
			}
		}
	}

	if err == nil {
		var ctor xt.ConnectorI
		var ctors []xt.ConnectorI
		ctor, err = xt.NewTcpConnector(regEnd)
		if err == nil {
			ctors = append(ctors, ctor)
			regPeer, err = xn.NewPeer(regName, regID, regCK, regSK,
				nil, ctors)
			if err == nil {
				_, err = node.AddPeer(regPeer)
			}
		}
	}
	if err == nil {
		cm = &xcl.ClusterMember{
			// Attrs gets negotiated
			ClusterName:    clusterName,
			ClusterAttrs:   clusterAttrs,
			ClusterID:      clusterID,
			ClusterMaxSize: size,
			EPCount:        epCount,
			// Members added on the fly
			Members: make([]*xcl.MemberInfo, size),
			Node:    *node,
		}
		mm = &MemberMaker{
			ProposedAttrs: attrs,
			DoneCh:        make(chan error, 1),
			RegPeer:       regPeer,
			ClusterMember: *cm,
		}
	}
	return
}

// Read the next message over the connection
func (mm *MemberMaker) readMsg() (m *XLRegMsg, err error) {
	inBuf, err := mm.ReadData()
	if err == nil && inBuf != nil {
		m, err = mm.DecryptUnpadDecode(inBuf)
	}
	return
}

// Write a message out over the connection
func (mm *MemberMaker) writeMsg(m *XLRegMsg) (err error) {
	var data []byte
	// serialize, marshal the message
	data, err = mm.EncodePadEncrypt(m)
	if err == nil {
		err = mm.WriteData(data)
	}
	return
}

// RUN CODE =========================================================

// Subclasses (UserMember, AdminClient, etc) use sequences of calls to
// these these functions to accomplish their purposes.

func (mm *MemberMaker) SessionSetup(proposedVersion uint32) (
	cnx *xt.TcpConnection, decidedVersion uint32, err error) {
	var (
		ciphertext1 []byte
		ciphertext2 []byte
	)
	// Set up connection to server. ---------------------------------
	ctor := mm.RegPeer.GetConnector(0)
	// DEBUG
	fmt.Printf("        SessionSetup: ctor is %s\n", ctor.String())
	// END
	var conn xt.ConnectionI
	//conn, err = ctor.Connect(xt.ANY_TCP_END_POINT) // 2016-11-14
	conn, err = ctor.Connect(nil)
	if err == nil {
		cnx = conn.(*xt.TcpConnection)
		// DEBUG
		fmt.Printf("        SessionSetup: cnx is %s\n", cnx.String())
		// END
		var cnxHandler *CnxHandler
		if err == nil {
			cnxHandler, err = NewCnxHandler(cnx, nil, nil)
			if err == nil {
				cnxHandler.State = MEMBER_START
				mm.CnxHandler = *cnxHandler
			}
		}
	}
	if err == nil {
		var cOneShot, cSession *xa.AesSession
		// Send HELLO -----------------------------------------------
		mm.Cnx = cnx
		ck := mm.RegPeer.GetCommsPublicKey()
		rng := xr.MakeSystemRNG()
		cOneShot, ciphertext1, err = xa.ClientEncryptHello(
			proposedVersion, ck, rng)
		if err == nil {
			err = mm.WriteData(ciphertext1)
			// Process HELLO REPLY ----------------------------------
			if err == nil {
				ciphertext2, err = mm.ReadData()

				if len(ciphertext2) == 0 {
					if err == nil {
						err = io.EOF
					}
				}
				if err == nil {
					cSession, decidedVersion,
						err = xa.ClientDecryptHelloReply(cOneShot, ciphertext2)
					if err == nil {
						mm.AesSession = *cSession
					}
				}
			}
		}
	}
	return
}

func (mm *MemberMaker) MemberAndOK() (err error) {

	var (
		ckBytes, skBytes []byte
		digSig           []byte
		hash             []byte
		id               []byte

		myEnds []string
	)
	// XXX attrs not actually dealt with

	node := mm.ClusterMember.Node
	nodeID := node.GetNodeID()
	if nodeID != nil {
		id = nodeID.Value()
	}
	name := node.GetName()
	ckPriv := node.GetCommsPrivateKey()
	skPriv := node.GetSigPrivateKey()

	// Send MEMBER MSG ==========================================
	// DEBUF
	fmt.Println("MemberMaker: have node info")
	// END
	aBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(aBytes, mm.ProposedAttrs)
	ckBytes, err = xc.RSAPubKeyToWire(&ckPriv.PublicKey)
	if err == nil {
		skBytes, err = xc.RSAPubKeyToWire(&skPriv.PublicKey)
		if err == nil {
			for i := 0; i < node.SizeEndPoints(); i++ {
				myEnd := node.GetEndPoint(i).String()
				if strings.HasPrefix(myEnd, "TcpEndPoint: ") {
					myEnd = myEnd[13:]
				}
				myEnds = append(myEnds, myEnd)
			}
			// calculate hash over fields in canonical order
			d := sha1.New()
			d.Write([]byte(name))
			d.Write(aBytes)
			d.Write(ckBytes)
			d.Write(skBytes)
			for i := 0; i < len(myEnds); i++ {
				d.Write([]byte(myEnds[i]))
			}
			hash = d.Sum(nil)
			// calculate digital signature
			digSig, err = rsa.SignPKCS1v15(rand.Reader, skPriv,
				crypto.SHA1, hash)
			// DEBUF
			//fmt.Printf("    MM: digSig added; error = %v\n", err)
			// END
		}
	}
	if err == nil {
		token := &XLRegMsg_Token{
			Name:     &name,
			ID:       id,
			Attrs:    &mm.ProposedAttrs,
			CommsKey: ckBytes,
			SigKey:   skBytes,
			MyEnds:   myEnds,
			DigSig:   digSig,
		}

		op := XLRegMsg_Member
		request := &XLRegMsg{
			Op: &op,
			// MemberName:  &name, // XXX redundant DROPPED
			MemberSpecs: token,
		}
		// SHOULD CHECK FOR TIMEOUT
		err = mm.writeMsg(request)
		// DEBUF
		//fmt.Printf("    MM: Member msg written; error = %v\n", err)
		// END
	}
	// Process CLIENT_OK --------------------------------------------
	// SHOULD CHECK FOR TIMEOUT
	response, err := mm.readMsg()
	if err == nil {
		// XXX Could check registry's notion of MemberID:
		//id := response.GetMemberID()
		//var nodeID *xi.NodeID
		//nodeID, err = xi.New(id)

		mm.Attrs = response.GetMemberAttrs()
	}
	// DEBUF
	//fmt.Printf("    MM: response received; error = %v\n", err)
	// END
	return
}

func (mm *MemberMaker) CreateAndReply() (err error) {

	var response *XLRegMsg

	// Send CREATE MSG ==========================================

	op := XLRegMsg_Create
	request := &XLRegMsg{
		Op:             &op,
		ClusterName:    &mm.ClusterName,
		ClusterAttrs:   &mm.ClusterAttrs,
		ClusterMaxSize: &mm.ClusterMaxSize,
		EndPointCount:  &mm.EPCount,
	}
	// SHOULD CHECK FOR TIMEOUT
	err = mm.writeMsg(request)
	if err == nil {
		// Process CREATE REPLY -------------------------------------
		// SHOULD CHECK FOR TIMEOUT AND VERIFY THAT IT'S A CREATE REPLY
		response, err = mm.readMsg()
		if err == io.EOF {
			err = nil
		}
		if err == nil {
			op = response.GetOp()
			_ = op
			if err == nil {
				id := response.GetClusterID()
				mm.ClusterID, err = xi.New(id)
				if err == nil {
					mm.ClusterAttrs = response.GetClusterAttrs()
					mm.ClusterMaxSize = response.GetClusterMaxSize()
				}
			}
		}
	}
	return
}

func (mm *MemberMaker) JoinAndReply() (err error) {

	// Send JOIN MSG ============================================
	op := XLRegMsg_Join
	id := mm.ClusterID.Value()
	request := &XLRegMsg{
		Op: &op,
		//ClusterName: &mm.ClusterName,
		ClusterID: id,
	}
	// SHOULD CHECK FOR TIMEOUT
	err = mm.writeMsg(request)

	// Process JOIN REPLY ---------------------------------------
	if err == nil {
		var response *XLRegMsg

		// SHOULD CHECK FOR TIMEOUT AND VERIFY THAT IT'S A JOIN REPLY
		response, err = mm.readMsg()
		op := response.GetOp()
		_ = op

		epCount := response.GetEndPointCount()
		if err == nil {
			clusterMaxSizeNow := response.GetClusterMaxSize()

			if mm.ClusterMaxSize != clusterMaxSizeNow {
				mm.ClusterMaxSize = clusterMaxSizeNow
				mm.Members = make([]*xcl.MemberInfo, mm.ClusterMaxSize)
			}
			mm.EPCount = epCount
			// This allows members knowing only the cluster name to
			// get the ID when they join
			id := response.GetClusterID()
			// mm.ClusterID, err = xi.New(id)
			_ = id // DO SOMETHING WITH IT  XXX
		}
	}
	return
}

// Collect information on all cluster members
func (mm *MemberMaker) GetAndMembers() (err error) {

	if mm.ClusterID == nil {
		fmt.Printf("** ENTERING GetAndMembers for %s with nil clusterID! **\n",
			mm.ClusterMember.Node.GetName())
	}
	const MAX_GET = 32 // 2014-01-31: was 16
	if mm.Members == nil {
		mm.Members = make([]*xcl.MemberInfo, mm.ClusterMaxSize)
	}
	stillToGet := xu.LowNMap(uint(mm.ClusterMaxSize))
	for count := 0; count < MAX_GET && stillToGet.Any(); count++ {

		var response *XLRegMsg

		for i := uint32(0); i < uint32(mm.ClusterMaxSize); i++ {
			if mm.Members[i] != nil {
				// XXX UNDESIRABLE CAST
				stillToGet = stillToGet.Clear(uint(i))
			}
		}

		// Send GET MSG =========================================
		op := XLRegMsg_GetCluster
		request := &XLRegMsg{
			Op:        &op,
			ClusterID: mm.ClusterID.Value(),
			Which:     &stillToGet.Bits,
		}
		// SHOULD CHECK FOR TIMEOUT
		err = mm.writeMsg(request)

		// Process MEMBERS = GET REPLY --------------------------
		if err != nil {
			break
		}
		response, err = mm.readMsg()
		if err != nil {
			break
		}
		op = response.GetOp()
		// XXX op MUST BE XLRegMsg_Members
		_ = op

		if err == nil {
			id := response.GetClusterID()
			_ = id // XXX ignore for now
			which := xu.NewBitMap64(response.GetWhich())
			tokens := response.GetTokens() // a slice
			if which.Any() {
				offset := 0
				for i := uint32(0); i < uint32(mm.ClusterMaxSize); i++ {
					// XXX UNDESIRABLE CAST
					if which.Test(uint(i)) {
						token := tokens[offset]
						offset++
						mm.Members[i], err = NewMemberInfoFromToken(
							token)
						if err == nil {
							// XXX UNDESIRABLE CAST
							stillToGet = stillToGet.Clear(uint(i))
						}
					}
				}
			}
			if stillToGet.None() {
				break
			}
			time.Sleep(50 * time.Millisecond) // WAS 10
		}
	}
	if err == nil {
		selfID := mm.GetNodeID().Value()

		for i := uint32(0); i < uint32(mm.ClusterMaxSize); i++ {

			mi := mm.Members[i]
			if mi == nil {
				continue
			}
			id := mi.Peer.GetNodeID()
			if id == nil {
				fmt.Printf("member has no nodeID!\n")
			}
			memberID := id.Value()
			if bytes.Equal(selfID, memberID) {
				mm.SelfIndex = i
				break
			}
		}
	}

	return
}

// Send Bye, wait for and process Ack.

func (mm *MemberMaker) ByeAndAck() (err error) {

	op := XLRegMsg_Bye
	request := &XLRegMsg{
		Op: &op,
	}
	// SHOULD CHECK FOR TIMEOUT
	err = mm.writeMsg(request)

	// Process ACK = BYE REPLY ----------------------------------
	if err == nil {
		var response *XLRegMsg

		// SHOULD CHECK FOR TIMEOUT AND VERIFY THAT IT'S AN ACK
		response, err = mm.readMsg()
		op := response.GetOp()
		_ = op
	}
	return
}
