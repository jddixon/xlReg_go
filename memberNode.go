package reg

// xlReg_go/memberNode.go

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	xc "github.com/jddixon/xlCrypto_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xa "github.com/jddixon/xlProtocol_go/aes_cnx"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
	xf "github.com/jddixon/xlUtil_go/lfs"
	"io/ioutil"
	"os"
	"path"
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

	AesCnxHandler

	// RegCred info: registry credentials -----------------
	RegName string
	RegID   *xi.NodeID
	RegCK   *rsa.PublicKey
	RegSK   *rsa.PublicKey
	RegEnd  xt.EndPointI

	// serverVersion xu.DecimalVersion		// missing

	regProtoVersion uint32 // protocol version used to talk to registry
	ClusterMember
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
		//node   *xn.Node
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

	var (
		config string
		// node   *xn.Node
	)

	// XXX check attrs, etc

	lfs := mn.ClusterMember.Node.GetLFS()
	pathToCfgDir := path.Join(lfs, ".xlattice")
	pathToCfgFile := path.Join(pathToCfgDir, "cluster.member.config")

	// DEBUG
	name := mn.ClusterMember.Node.GetName()
	fmt.Printf("member %-8s: config file is %s\n", name, pathToCfgFile)
	// END

	_, err = os.Stat(pathToCfgDir)
	if os.IsNotExist(err) {
		err = xf.CheckLFS(pathToCfgDir, 0740)
	} else if err != nil {
		// DEBUG
		fmt.Printf("  member %-8s: error from Stat %s is %v\n",
			name, pathToCfgFile, err)
		// END
	}

	if err == nil {
		// DEBUG
		//fmt.Printf("  member %-8s: serializing\n", mn.Name)
		// END
		// mn.Node = *node
		config = mn.ClusterMember.String()
		if err == nil {
			// DEBUG
			//fmt.Printf("  member %-8s: writing config file\n", mn.Name)
			// END
			err = ioutil.WriteFile(pathToCfgFile, []byte(config), 0600)
		}
	}
	// DEBUG
	//if err != nil {
	//	fmt.Printf("  member %-8s: ERROR %s\n", mn.Name, err.Error())
	//}
	// END
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
		cm      *ClusterMember
		isAdmin = (attrs & ATTR_ADMIN) != 0
	)

	// DEBUG
	name := node.GetName()
	fmt.Printf("NewMemberMaker name %s, attrs 0x%x, epCount = %d\n",
		name, attrs, epCount)
	// END

	// sanity checks on parameter list
	if node == nil {
		err = MissingNode
	} else {
		if regName == "" || regID == nil || regEnd == nil ||
			regCK == nil {

			err = MissingServerInfo
		}
		if err == nil {
			if (attrs & ATTR_SOLO) == uint64(0) {
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
		cnxHandler := &AesCnxHandler{State: MEMBER_START}
		cm = &ClusterMember{
			// Attrs gets negotiated
			ClusterName:  clusterName,
			ClusterAttrs: clusterAttrs,
			ClusterID:    clusterID,
			ClusterSize:  size,
			EPCount:      epCount,
			// Members added on the fly
			Members: make([]*MemberInfo, size),
			Node:    *node,
		}
		mm = &MemberMaker{
			ProposedAttrs: attrs,
			DoneCh:        make(chan error, 1),
			RegName:       regName,
			RegID:         regID,
			RegEnd:        regEnd,
			RegCK:         regCK,
			RegSK:         regSK,
			AesCnxHandler: *cnxHandler,
			ClusterMember: *cm,
		}
	}
	return
}

// Read the next message over the connection
func (mm *MemberMaker) readMsg() (m *XLRegMsg, err error) {
	inBuf, err := mm.ReadData()
	if err == nil && inBuf != nil {
		m, err = DecryptUnpadDecode(inBuf, mm.decrypter)
	}
	return
}

// Write a message out over the connection
func (mm *MemberMaker) writeMsg(m *XLRegMsg) (err error) {
	var data []byte
	// serialize, marshal the message
	data, err = EncodePadEncrypt(m, mm.encrypter)
	if err == nil {
		err = mm.WriteData(data)
	}
	return
}

// RUN CODE =========================================================

// Subclasses (UserMember, AdminMember, etc) use sequences of calls to
// these these functions to accomplish their purposes.

func (mm *MemberMaker) SessionSetup(proposedVersion uint32) (
	cnx *xt.TcpConnection, decidedVersion uint32, err error) {
	var (
		ciphertext1, iv1, key1, salt1, salt1c []byte
		ciphertext2, iv2, key2, salt2         []byte
	)
	// Set up connection to server. -----------------------------
	ctor, err := xt.NewTcpConnector(mm.RegEnd)
	if err == nil {
		var conn xt.ConnectionI
		conn, err = ctor.Connect(nil)
		if err == nil {
			cnx = conn.(*xt.TcpConnection)
		}
	}
	// Send HELLO -----------------------------------------------
	if err == nil {
		mm.Cnx = cnx
		ciphertext1, iv1, key1, salt1,
			err = xa.ClientEncodeHello(proposedVersion, mm.RegCK)
	}
	if err == nil {
		err = mm.WriteData(ciphertext1)
	}
	// Process HELLO REPLY --------------------------------------
	if err == nil {
		ciphertext2, err = mm.ReadData()
	}
	if err == nil {
		iv2, key2, salt2, salt1c, decidedVersion,
			err = xa.ClientDecodeHelloReply(ciphertext2, iv1, key1)
		_ = salt1c // XXX
	}
	// Set up AES engines ---------------------------------------
	if err == nil {
		mm.salt1 = salt1
		mm.iv2 = iv2
		mm.key2 = key2
		mm.salt2 = salt2
		mm.regProtoVersion = decidedVersion
		mm.engine, err = aes.NewCipher(key2)
		if err == nil {
			mm.encrypter = cipher.NewCBCEncrypter(mm.engine, iv2)
			mm.decrypter = cipher.NewCBCDecrypter(mm.engine, iv2)
		}
	}
	return
}

func (mm *MemberMaker) MemberAndOK() (err error) {

	var (
		ckBytes, skBytes []byte
		digSig           []byte
		hash             []byte
		myEnds           []string
	)
	// XXX attrs not actually dealt with

	node := mm.ClusterMember.Node
	name := node.GetName()
	ckPriv := node.GetCommsPrivateKey()
	skPriv := node.GetSigPrivateKey()

	// Send MEMBER MSG ==========================================
	aBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(aBytes, mm.ProposedAttrs)
	ckBytes, err = xc.RSAPubKeyToWire(&ckPriv.PublicKey)
	if err == nil {
		skBytes, err = xc.RSAPubKeyToWire(&skPriv.PublicKey)
		if err == nil {
			for i := 0; i < node.SizeEndPoints(); i++ {
				myEnds = append(myEnds, node.GetEndPoint(i).String())
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
		}
	}
	if err == nil {
		token := &XLRegMsg_Token{
			Name:     &name,
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
	return
}

func (mm *MemberMaker) CreateAndReply() (err error) {

	var response *XLRegMsg

	// Send CREATE MSG ==========================================

	op := XLRegMsg_Create
	request := &XLRegMsg{
		Op:            &op,
		ClusterName:   &mm.ClusterName,
		ClusterAttrs:  &mm.ClusterAttrs,
		ClusterSize:   &mm.ClusterSize,
		EndPointCount: &mm.EPCount,
	}
	// SHOULD CHECK FOR TIMEOUT
	err = mm.writeMsg(request)
	if err == nil {
		// Process CREATE REPLY -------------------------------------
		// SHOULD CHECK FOR TIMEOUT AND VERIFY THAT IT'S A CREATE REPLY
		response, err = mm.readMsg()
		op = response.GetOp()
		_ = op
		if err == nil {
			id := response.GetClusterID()
			mm.ClusterID, err = xi.New(id)
			mm.ClusterAttrs = response.GetClusterAttrs()
			mm.ClusterSize = response.GetClusterSize()
			// XXX no check on err
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
			clusterSizeNow := response.GetClusterSize()

			if mm.ClusterSize != clusterSizeNow {
				mm.ClusterSize = clusterSizeNow
				mm.Members = make([]*MemberInfo, mm.ClusterSize)
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
		mm.Members = make([]*MemberInfo, mm.ClusterSize)
	}
	stillToGet := xu.LowNMap(uint(mm.ClusterSize))
	for count := 0; count < MAX_GET && stillToGet.Any(); count++ {

		var response *XLRegMsg

		for i := uint32(0); i < uint32(mm.ClusterSize); i++ {
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
				for i := uint32(0); i < uint32(mm.ClusterSize); i++ {
					// XXX UNDESIRABLE CAST
					if which.Test(uint(i)) {
						token := tokens[offset]
						offset++
						mm.Members[i], err = NewMemberInfoFromToken(
							token)
						// XXX UNDESIRABLE CAST
						stillToGet = stillToGet.Clear(uint(i))
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
		selfID := mm.RegID.Value()

		for i := uint32(0); i < uint32(mm.ClusterSize); i++ {

			mi := mm.Members[i]
			if mi == nil {
				// DEBUG
				msg := fmt.Sprintf("cluster member %d is nil", i)
				fmt.Println(msg)
				// END
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
		// DEBUG
	} else {
		fmt.Printf("mm.GetAndMembers: error '%s'\n", err.Error())
		// END
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
