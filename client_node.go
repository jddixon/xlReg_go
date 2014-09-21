package reg

// xlReg_go/client_node.go  WILL BECOME memberNode.go

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

type MemberNode struct {
	DoneCh          chan error
	ProposedAttrs   uint64
	ProposedVersion uint32 // proposed by member

	ckPriv, skPriv *rsa.PrivateKey
	AesCnxHandler

	// RegCred info: registry credentials ---------------------------
	RegName string
	RegID   *xi.NodeID
	RegCK   *rsa.PublicKey
	RegSK   *rsa.PublicKey
	RegEnd  xt.EndPointI
	// serverVersion xu.DecimalVersion		// missing

	regProtoVersion uint32 // protocol version used to talk to registry

	// REDUNDANT: INFORMATION USED TO BUILD NODE ====================
	// This is used to build the node and so is persisted as part of
	// the node when that is saved.
	EndPoints []xt.EndPointI
	LFS       string
	Name      string
	MemberID  *xi.NodeID

	ClusterMember
}

// Returns a copy of the node's comms private RSA key
func (cn *MemberNode) GetCKPriv() rsa.PrivateKey {
	return *cn.ckPriv
}

// Returns a copy of the node's sig private RSA key
func (cn *MemberNode) GetSKPriv() rsa.PrivateKey {
	return *cn.skPriv
}

// Create just the Node for this member and write it to the conventional
// place in the file system.
func (cn *MemberNode) PersistNode() (err error) {

	var (
		config string
		node   *xn.Node
	)

	// XXX check attrs, etc
	pathToCfgDir := path.Join(cn.LFS, ".xlattice")
	pathToCfgFile := path.Join(pathToCfgDir, "node.config")
	found, err := xf.PathExists(pathToCfgDir)
	if err == nil && !found {
		err = os.MkdirAll(pathToCfgDir, 0740)
	}
	if err == nil {
		node, err = xn.New(cn.Name, cn.MemberID, cn.LFS,
			cn.ckPriv, cn.skPriv, nil, cn.EndPoints, nil)
	}
	if err == nil {
		cn.Node = *node
		config = node.String()
	}
	if err == nil {
		err = ioutil.WriteFile(pathToCfgFile, []byte(config), 0600)
	}
	return
} // FOO

// Create the Node for this client and write the serialized ClusterMember
// to the conventional place in the file system.
func (cn *MemberNode) PersistClusterMember() (err error) {

	var (
		config string
		node   *xn.Node
	)

	// XXX check attrs, etc
	pathToCfgDir := path.Join(cn.LFS, ".xlattice")
	pathToCfgFile := path.Join(pathToCfgDir, "cluster.member.config")
	found, err := xf.PathExists(pathToCfgDir)
	if err == nil && !found {
		err = os.MkdirAll(pathToCfgDir, 0740)
	}
	if err == nil {
		node, err = xn.New(cn.Name, cn.MemberID, cn.LFS,
			cn.ckPriv, cn.skPriv, nil, cn.EndPoints, nil)
	}
	if err == nil {
		cn.Node = *node
		config = cn.ClusterMember.String() // XXX sometimes panics
	}
	if err == nil {
		err = ioutil.WriteFile(pathToCfgFile, []byte(config), 0600)
	}
	return
}

// Given contact information for a registry and the name of a cluster,
// the client joins the cluster, collects information on the other members,
// and terminates when it has info on the entire membership.

func NewMemberNode(
	name, lfs string, ckPriv, skPriv *rsa.PrivateKey, attrs uint64,
	regName string, regID *xi.NodeID, regEnd xt.EndPointI,
	regCK, regSK *rsa.PublicKey,
	clusterName string, clusterAttrs uint64, clusterID *xi.NodeID,
	size, epCount uint32, e []xt.EndPointI) (
	cn *MemberNode, err error) {

	var (
		cm      *ClusterMember
		isAdmin = (attrs & ATTR_ADMIN) != 0
		node    *xn.Node
	)

	// sanity checks on parameter list
	if regName == "" || regID == nil || regEnd == nil ||
		regCK == nil {

		err = MissingServerInfo
	}
	if attrs&ATTR_SOLO == uint64(0) {
		if err == nil && clusterName == "" {
			err = MissingClusterNameOrID
		}
		if err == nil && size < uint32(1) {
			// err = ClusterMustHaveTwo
			err = ClusterMustHaveMember
		}
	}
	if err == nil {
		// if the client is an admin client epCount applies to the cluster
		if epCount < uint32(1) {
			epCount = uint32(1)
		}
		if !isAdmin {
			// XXX There is some confusion here: we don't require that
			// all members have the same number of endpoints
			actualEPCount := uint32(len(e))
			if actualEPCount == 0 {
				err = MemberMustHaveEndPoint
			} else if epCount > actualEPCount {
				epCount = actualEPCount
			}
		}
		if err == nil && ckPriv == nil {
			ckPriv, err = rsa.GenerateKey(rand.Reader, 2048)
		}
		if err == nil && skPriv == nil {
			skPriv, err = rsa.GenerateKey(rand.Reader, 2048)
		}
	}

	if err == nil && node == nil && (ckPriv == nil || skPriv == nil) {
		err = NoNodeNoKeys
	}
	if err == nil {
		cnxHandler := &AesCnxHandler{State: MEMBER_START}
		cm = &ClusterMember{
			// Attrs gets negotiated
			ClusterName:  clusterName,
			ClusterAttrs: clusterAttrs,
			ClusterID:    clusterID,
			ClusterSize:  size,
			EpCount:      epCount,
			// Members added on the fly
			Members: make([]*MemberInfo, size),

			// Node NOT YET INITIALIZED
		}
		cn = &MemberNode{
			Name:          name,
			LFS:           lfs, // if blank, node is ephemeral
			ProposedAttrs: attrs,
			DoneCh:        make(chan error, 1),
			RegName:       regName,
			RegID:         regID,
			RegEnd:        regEnd,
			RegCK:         regCK,
			RegSK:         regSK,
			EndPoints:     e,

			ckPriv:        ckPriv,
			skPriv:        skPriv,
			AesCnxHandler: *cnxHandler,

			ClusterMember: *cm,
		}
	}
	return
}

// Read the next message over the connection
func (cn *MemberNode) readMsg() (m *XLRegMsg, err error) {
	inBuf, err := cn.ReadData()
	if err == nil && inBuf != nil {
		m, err = DecryptUnpadDecode(inBuf, cn.decrypter)
	}
	return
}

// Write a message out over the connection
func (cn *MemberNode) writeMsg(m *XLRegMsg) (err error) {
	var data []byte
	// serialize, marshal the message
	data, err = EncodePadEncrypt(m, cn.encrypter)
	if err == nil {
		err = cn.WriteData(data)
	}
	return
}

// RUN CODE =========================================================

// Subclasses (UserMember, AdminMember, etc) use sequences of calls to
// these these functions to accomplish their purposes.

func (cn *MemberNode) SessionSetup(proposedVersion uint32) (
	cnx *xt.TcpConnection, decidedVersion uint32, err error) {
	var (
		ciphertext1, iv1, key1, salt1, salt1c []byte
		ciphertext2, iv2, key2, salt2         []byte
	)
	// Set up connection to server. -----------------------------
	ctor, err := xt.NewTcpConnector(cn.RegEnd)
	if err == nil {
		var conn xt.ConnectionI
		conn, err = ctor.Connect(nil)
		if err == nil {
			cnx = conn.(*xt.TcpConnection)
		}
	}
	// Send HELLO -----------------------------------------------
	if err == nil {
		cn.Cnx = cnx
		ciphertext1, iv1, key1, salt1,
			err = xa.ClientEncodeHello(proposedVersion, cn.RegCK)
	}
	if err == nil {
		err = cn.WriteData(ciphertext1)
	}
	// Process HELLO REPLY --------------------------------------
	if err == nil {
		ciphertext2, err = cn.ReadData()
	}
	if err == nil {
		iv2, key2, salt2, salt1c, decidedVersion,
			err = xa.ClientDecodeHelloReply(ciphertext2, iv1, key1)
		_ = salt1c // XXX
	}
	// Set up AES engines ---------------------------------------
	if err == nil {
		cn.salt1 = salt1
		cn.iv2 = iv2
		cn.key2 = key2
		cn.salt2 = salt2
		cn.regProtoVersion = decidedVersion
		cn.engine, err = aes.NewCipher(key2)
		if err == nil {
			cn.encrypter = cipher.NewCBCEncrypter(cn.engine, iv2)
			cn.decrypter = cipher.NewCBCDecrypter(cn.engine, iv2)
		}
	}
	return
}

func (cn *MemberNode) MemberAndOK() (err error) {

	var (
		ckBytes, skBytes []byte
		digSig           []byte
		hash             []byte
		myEnds           []string
	)
	// XXX attrs not actually dealt with

	// Send MEMBER MSG ==========================================
	aBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(aBytes, cn.ProposedAttrs)
	ckBytes, err = xc.RSAPubKeyToWire(&cn.ckPriv.PublicKey)
	if err == nil {
		skBytes, err = xc.RSAPubKeyToWire(&cn.skPriv.PublicKey)
		if err == nil {
			for i := 0; i < len(cn.EndPoints); i++ {
				myEnds = append(myEnds, cn.EndPoints[i].String())
			}
			// calculate hash over fields in canonical order
			d := sha1.New()
			d.Write([]byte(cn.Name))
			d.Write(aBytes)
			d.Write(ckBytes)
			d.Write(skBytes)
			for i := 0; i < len(myEnds); i++ {
				d.Write([]byte(myEnds[i]))
			}
			hash = d.Sum(nil)
			// calculate digital signature
			digSig, err = rsa.SignPKCS1v15(rand.Reader, cn.skPriv,
				crypto.SHA1, hash)
		}
	}
	if err == nil {
		token := &XLRegMsg_Token{
			Name:     &cn.Name,
			Attrs:    &cn.ProposedAttrs,
			CommsKey: ckBytes,
			SigKey:   skBytes,
			MyEnds:   myEnds,
			DigSig:   digSig,
		}

		op := XLRegMsg_Member
		request := &XLRegMsg{
			Op: &op,
			// MemberName:  &cn.Name, // XXX redundant DROPPED
			MemberSpecs: token,
		}
		// SHOULD CHECK FOR TIMEOUT
		err = cn.writeMsg(request)
	}
	// Process CLIENT_OK --------------------------------------------
	// SHOULD CHECK FOR TIMEOUT
	response, err := cn.readMsg()
	if err == nil {
		id := response.GetMemberID()
		var nodeID *xi.NodeID
		nodeID, err = xi.New(id)
		if err == nil {
			cn.MemberID = nodeID
			cn.Attrs = response.GetMemberAttrs()
		}
	}
	return
}

func (cn *MemberNode) CreateAndReply() (err error) {

	var response *XLRegMsg

	// Send CREATE MSG ==========================================

	op := XLRegMsg_Create
	request := &XLRegMsg{
		Op:            &op,
		ClusterName:   &cn.ClusterName,
		ClusterAttrs:  &cn.ClusterAttrs,
		ClusterSize:   &cn.ClusterSize,
		EndPointCount: &cn.EpCount,
	}
	// SHOULD CHECK FOR TIMEOUT
	err = cn.writeMsg(request)
	if err == nil {
		// Process CREATE REPLY -------------------------------------
		// SHOULD CHECK FOR TIMEOUT AND VERIFY THAT IT'S A CREATE REPLY
		response, err = cn.readMsg()
		op = response.GetOp()
		_ = op
		if err == nil {
			id := response.GetClusterID()
			cn.ClusterID, err = xi.New(id)
			cn.ClusterAttrs = response.GetClusterAttrs()
			cn.ClusterSize = response.GetClusterSize()
			// XXX no check on err
		}
	}
	return
}

func (cn *MemberNode) JoinAndReply() (err error) {

	// Send JOIN MSG ============================================
	op := XLRegMsg_Join
	id := cn.ClusterID.Value()
	request := &XLRegMsg{
		Op: &op,
		//ClusterName: &cn.ClusterName,
		ClusterID: id,
	}
	// SHOULD CHECK FOR TIMEOUT
	err = cn.writeMsg(request)

	// Process JOIN REPLY ---------------------------------------
	if err == nil {
		var response *XLRegMsg

		// SHOULD CHECK FOR TIMEOUT AND VERIFY THAT IT'S A JOIN REPLY
		response, err = cn.readMsg()
		op := response.GetOp()
		_ = op

		epCount := response.GetEndPointCount()
		if err == nil {
			clusterSizeNow := response.GetClusterSize()

			if cn.ClusterSize != clusterSizeNow {
				cn.ClusterSize = clusterSizeNow
				cn.Members = make([]*MemberInfo, cn.ClusterSize)
			}
			cn.EpCount = epCount
			// This allows members knowing only the cluster name to
			// get the ID when they join
			id := response.GetClusterID()
			// cn.ClusterID, err = xi.New(id)
			_ = id // DO SOMETHING WITH IT  XXX
		}
	}
	return
}

// Collect information on all cluster members
func (cn *MemberNode) GetAndMembers() (err error) {

	if cn.ClusterID == nil {
		fmt.Printf("** ENTERING GetAndMembers for %s with nil clusterID! **\n",
			cn.Name)
	}
	const MAX_GET = 32 // 2014-01-31: was 16
	if cn.Members == nil {
		cn.Members = make([]*MemberInfo, cn.ClusterSize)
	}
	stillToGet := xu.LowNMap(uint(cn.ClusterSize))
	for count := 0; count < MAX_GET && stillToGet.Any(); count++ {

		var response *XLRegMsg

		for i := uint32(0); i < uint32(cn.ClusterSize); i++ {
			if cn.Members[i] != nil {
				// XXX UNDESIRABLE CAST
				stillToGet = stillToGet.Clear(uint(i))
			}
		}

		// Send GET MSG =========================================
		op := XLRegMsg_GetCluster
		request := &XLRegMsg{
			Op:        &op,
			ClusterID: cn.ClusterID.Value(),
			Which:     &stillToGet.Bits,
		}
		// SHOULD CHECK FOR TIMEOUT
		err = cn.writeMsg(request)

		// Process MEMBERS = GET REPLY --------------------------
		if err != nil {
			break
		}
		response, err = cn.readMsg()
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
				for i := uint32(0); i < uint32(cn.ClusterSize); i++ {
					// XXX UNDESIRABLE CAST
					if which.Test(uint(i)) {
						token := tokens[offset]
						offset++
						cn.Members[i], err = NewMemberInfoFromToken(
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
		selfID := cn.RegID.Value()

		for i := uint32(0); i < uint32(cn.ClusterSize); i++ {

			member := cn.Members[i]
			if member == nil {
				msg := fmt.Sprintf("cluster member %d is nil", i)
				// DEBUG
				fmt.Println(msg)
				// END
				continue
			}

			// XXX NPE HERE if member is nil
			id := member.GetNodeID()
			if id == nil {
				fmt.Printf("member has no nodeID!\n")
			}
			memberID := id.Value()
			// END HERE ///////////////////////////////////

			if bytes.Equal(selfID, memberID) {
				cn.SelfIndex = i
				break
			}
		}
		// DEBUG
	} else {
		fmt.Printf("cn.GetAndMembers: error '%s'\n", err.Error())
		// END
	}

	return
}

// Send Bye, wait for and process Ack.

func (cn *MemberNode) ByeAndAck() (err error) {

	op := XLRegMsg_Bye
	request := &XLRegMsg{
		Op: &op,
	}
	// SHOULD CHECK FOR TIMEOUT
	err = cn.writeMsg(request)

	// Process ACK = BYE REPLY ----------------------------------
	if err == nil {
		var response *XLRegMsg

		// SHOULD CHECK FOR TIMEOUT AND VERIFY THAT IT'S AN ACK
		response, err = cn.readMsg()
		op := response.GetOp()
		_ = op
	}
	return
}
