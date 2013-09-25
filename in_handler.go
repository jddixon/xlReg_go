package reg

// xlattice_go/reg/in_handler.go

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	// xc "github.com/jddixon/xlattice_go/crypto"
	"github.com/jddixon/xlattice_go/msg"
	// xi "github.com/jddixon/xlattice_go/nodeID"
	xt "github.com/jddixon/xlattice_go/transport"
)

var _ = fmt.Print

const (
	// States through which the input cnx may pass
	HELLO_RCVD = iota
	CLIENT_DETAILS_RCVD
	CREATE_REQUEST_RCVD
	JOIN_RCVD

	// Once the connection has reached this state, no more messages
	// can be accepted.
	BYE_RCVD

	// When we reach this state, the connection must be closed.
	IN_CLOSED			
)

const (
	// the number of valid states upon receiving a message from a client
	IN_STATE_COUNT		= BYE_RCVD + 1 

	// the tags that InHandler will accept from a client
	MIN_TAG				= 0
	MAX_TAG				= 4
	
	MSG_HANDLER_COUNT	= MAX_TAG + 1
)

var (
	msgHandlers [][]interface{}
)
func init() {
	// msgHandlers = make([][]interface{}, BYE_RCVD, MSG_HANDLER_COUNT)
	
	msgHandlers = [][]interface{}{
		// client messages permitted in HELLO_RCVD state
		{doClientMsg, badCombo,    badCombo,  badCombo, doByeMsg},
		// messages permitted in CLIENT_DETAILS_RCVD
		{badCombo,    doCreateMsg, doJoinMsg, badCombo, doByeMsg},
		// messages permitted in CREATE_REQUEST_RCVD
		{badCombo,    badCombo,    doJoinMsg, badCombo, doByeMsg},
		// messages permitted in JOIN_RCVD
		{badCombo,    badCombo,    badCombo,  doGetMsg, doByeMsg},
	}

}
type InHandler struct {
	iv1, key1, iv2, key2, salt1, salt2 []byte
	engineS            cipher.Block
	encrypterS         cipher.BlockMode
	decrypterS         cipher.BlockMode
	reg                *Registry
	thisMember         *ClusterMember
	clusterName        string
	clusterID          []byte
	clusterSize        int
	version            uint32 // protocol version used in session
	known              uint64 // a bit vector:
	entryState         int
	exitState          int
	msgIn              *XLRegMsg
	msgOut             *XLRegMsg
	errOut             error
	CnxHandler
}

// Given an open new connection, create a handler for the connection,
// associating the connection with a registry.

func NewInHandler(reg *Registry, conn xt.ConnectionI) (
	h *InHandler, err error) {

	if reg == nil {
		return nil, NilRegistry
	}
	rn := &reg.RegNode
	if rn == nil {
		err = msg.NilNode
	} else if conn == nil {
		err = msg.NilConnection
	} else {
		cnx := conn.(*xt.TcpConnection)
		h = &InHandler{
			CnxHandler: CnxHandler{
				Cnx: cnx,
			},
		}
	}
	return
}

func SetUpSessionKey(h *InHandler) (err error) {
	h.engineS, err = aes.NewCipher(h.key2)
	if err == nil {
		h.encrypterS = cipher.NewCBCEncrypter(h.engineS, h.iv2)
		h.decrypterS = cipher.NewCBCDecrypter(h.engineS, h.iv2)
	}
	return
}

// Convert a protobuf op into a zero-based tag for use in the InHandler's
// dispatch table.
func op2tag(op XLRegMsg_Tag) int {
	return int(op - XLRegMsg_Client) / 2
}

// Given a handler associating an open new connection with a registry, 
// process a hello message for this node, which creates a session.
// The hello message contains an AES Key+IV, a salt, and a requested 
// protocol version. The salt must be at least eight bytes long.

func (h *InHandler) Run() (err error) {

	defer func() {
		if h.Cnx != nil {
			h.Cnx.Close()
		}
	}()

	// This adds an AES iv2 and key2 to the handler.
	err = handleHello(h) 
	if err != nil {
		return
	}
	// Given iv2, key2 create encrypt and decrypt engines.
	err = SetUpSessionKey(h)
	if err != nil {
		return
	}
	for {
		// REQUEST --------------------------------------------------
		//   receive the raw data off the wire
		var ciphertext []byte
		ciphertext, err = h.readData()
		if err != nil {
			return
		}
		h.msgIn, err = DecryptUnpadDecode(ciphertext, h.decrypterS)
		if err != nil {
			return
		}
		op := h.msgIn.GetOp() 
	
		// XXX THIS SHOULD BE A RANGE CHECK.  Unfortunately the 
		// valid ops are currently 3, 5, 7, 9, 13, 15 
		if op != XLRegMsg_Client {
			err = UnexpectedMsgType
			return
		}
		tag := op2tag(op)
		
		// CRUDE CHECK
		if tag != 0 {
			fmt.Printf("InHandler.Run(): tag is %d\n", tag)
		}

		// ACTION ----------------------------------------------------
		// Take the action appropriate for the current state
		msgHandlers[h.entryState][tag].(func(*InHandler))(h)

		// RESPONSE -------------------------------------------------
		// Convert any error encountered into an error message to be
		// sent to the client.
		if h.errOut != nil {
			op := XLRegMsg_Error
			s  := h.errOut.Error()
			h.msgOut = &XLRegMsg{
				Op:			&op,
				ErrDesc:	&s,
			}
			h.errOut = nil			// reduce potential for confusion
			h.exitState = IN_CLOSED	// there is no recovery from errors
		}

		// encode, pad, and encrypt the XLRegMsg object 
		if h.msgOut != nil {
			ciphertext, err = EncodePadEncrypt(h.msgOut, h.encrypterS)	

			// XXX log any error

			// put the ciphertext on the wire
			if err == nil {
				err = h.writeData(ciphertext)

				// XXX log any error
			}
		}
		h.entryState = h.exitState
		if h.exitState == IN_CLOSED {
			break
		} 
	}

	return
}

/////////////////////////////////////////////////////////////////////
// RSA-BASED MESSAGE PAIR
/////////////////////////////////////////////////////////////////////

// The client has sent the server a one-time AES key+iv encrypted with
// the server's RSA comms public key.  The server creates the real
// session iv+key and returns them to the client encrypted with the
// one-time key+iv.

func handleHello(h *InHandler) (err error) {
	var (
		ciphertext, iv1, key1, salt1 []byte
		version1                     uint32
	)
	rn := &h.reg.RegNode
	ciphertext, err = h.readData()
	if err == nil {
		iv1, key1, salt1, version1,
			err = msg.ServerDecodeHello(ciphertext, rn.ckPriv)
	}
	if err == nil {
		version2 := version1 // accept whatever version they propose
		iv2, key2, salt2, ciphertextOut, err := msg.ServerEncodeHelloReply(
			iv1, key1, salt1, version2)
		if err == nil {
			err = h.writeData(ciphertextOut)
		}
		if err == nil {
			h.iv1 = iv1
			h.key1 = key1
			h.iv2 = iv2
			h.key2 = key2
			h.salt1 = salt1
			h.salt2 = salt2
			h.version = version2
			h.State = HELLO_RCVD
		}
	}
	// On any error silently close the connection and delete the handler,
	// an exciting thing to do.
	if err != nil {
		h.Cnx.Close()
		h = nil
	}
	return
}
