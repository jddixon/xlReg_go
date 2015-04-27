package reg

// xlReg_go/in_handler.go

import (
	"fmt"
	xa "github.com/jddixon/xlProtocol_go/aes_cnx"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
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
	IN_STATE_COUNT = BYE_RCVD + 1

	// the tags that InHandler will accept from a client
	MIN_TAG = 0
	MAX_TAG = 4

	MSG_HANDLER_COUNT = MAX_TAG + 1
)

var (
	msgHandlers   [][]interface{}
	serverVersion xu.DecimalVersion
)

func init() {
	// msgHandlers = make([][]interface{}, BYE_RCVD, MSG_HANDLER_COUNT)

	msgHandlers = [][]interface{}{
		// client messages permitted in HELLO_RCVD state
		{doClientMsg, badCombo, badCombo, badCombo, doByeMsg},
		// messages permitted in CLIENT_DETAILS_RCVD
		{badCombo, doCreateMsg, doJoinMsg, badCombo, doByeMsg},
		// messages permitted in CREATE_REQUEST_RCVD
		{badCombo, badCombo, doJoinMsg, badCombo, doByeMsg},
		// messages permitted in JOIN_RCVD
		{badCombo, badCombo, badCombo, doGetMsg, doByeMsg},
	}
	var err error
	serverVersion, err = xu.ParseDecimalVersion(VERSION)
	if err != nil {
		panic(err)
	}
}

/**
 * Handler for connections incoming to the registry server from clients.
 */
type InHandler struct {
	reg        *Registry
	thisClient *ClientInfo
	cluster    *RegCluster
	version    uint32 // protocol version used in session
	known      uint64 // a bit vector:
	entryState int
	exitState  int
	msgIn      *XLRegMsg
	msgOut     *XLRegMsg
	errOut     error

	CnxHandler
}

// Given an open new connection, create a handler for the connection,
// associating the connection with a registry.

func NewInHandler(reg *Registry, conn xt.ConnectionI) (
	h *InHandler, err error) {

	var cnxHandler *CnxHandler
	if reg == nil {
		return nil, NilRegistry
	}
	rn := &reg.RegNode
	if rn == nil {
		err = xa.NilNode
	} else if conn == nil {
		err = xa.NilConnection
	}
	if err == nil {
		cnx := conn.(*xt.TcpConnection)
		cnxHandler, err = NewCnxHandler(cnx, nil, nil)
	}
	if err == nil {
		h = &InHandler{
			reg:        reg,
			CnxHandler: *cnxHandler,
		}
	}
	return
}

// Convert a protobuf op into a zero-based tag for use in the InHandler's
// dispatch table.
func op2tag(op XLRegMsg_Tag) int {
	return int(op-XLRegMsg_Member) / 2
}

// Given a handler associating an open new connection with a registry,
// process a hello message for this node, which creates a session.
// The hello message contains an AES Key, a salt, and a requested
// protocol version. The salt must be at least eight bytes long.

func (h *InHandler) Start() (err error) {

	defer func() {
		if h.Cnx != nil {
			h.Cnx.Close()
		}
	}()

	// This adds an AES key2 to the handler.
	err = handleHello(h)
	if err != nil {
		return
	}
	for {
		var (
			tag int
		)
		// REQUEST --------------------------------------------------
		//   receive the raw data off the wire
		var ciphertext []byte
		ciphertext, err = h.ReadData()
		if err != nil {
			return
		}
		h.msgIn, err = h.DecryptUnpadDecode(ciphertext)
		if err != nil {
			return
		}
		op := h.msgIn.GetOp()
		// TODO: range check on either op or tag
		tag = op2tag(op)
		if tag < MIN_TAG || tag > MAX_TAG {
			h.errOut = TagOutOfRange
		}
		// ACTION ----------------------------------------------------
		// Take the action appropriate for the current state
		msgHandlers[h.entryState][tag].(func(*InHandler))(h)

		// RESPONSE -------------------------------------------------
		// Convert any error encountered into an error message to be
		// sent to the client.
		if h.errOut != nil {
			h.reg.Logger.Printf("errOut to client: %s\n", h.errOut.Error())

			op := XLRegMsg_Error
			s := h.errOut.Error()
			h.msgOut = &XLRegMsg{
				Op:      &op,
				ErrDesc: &s,
			}
			h.errOut = nil          // reduce potential for confusion
			h.exitState = IN_CLOSED // there is no recovery from errors
		}

		// encode, pad, and encrypt the XLRegMsg object
		if h.msgOut != nil {
			ciphertext, err = h.EncodePadEncrypt(h.msgOut)

			// XXX log any error
			if err != nil {
				h.reg.Logger.Printf(
					"InHandler.Start: EncodePadEncrypt returns %v\n", err)
			}

			// put the ciphertext on the wire
			if err == nil {
				err = h.WriteData(ciphertext)

				// XXX log any error
				if err != nil {
					h.reg.Logger.Printf(
						"InHandler.Start: WriteData returns %v\n", err)
				}
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

// The client has sent the server a one-time AES key encrypted with
// the server's RSA comms public key.  The server creates the real
// session key and returns it to the client encrypted with the
// one-time key.

func handleHello(h *InHandler) (err error) {
	var (
		sOneShot   *xa.AesSession
		ciphertext []byte
		version1   uint32
	)
	// DEBUG
	fmt.Println("entering handleHello")
	// END
	rn := &h.reg.RegNode
	ciphertext, err = h.ReadData()
	if err == nil {
		sOneShot, version1,
			err = xa.ServerDecryptHello(ciphertext, rn.ckPriv, h.RNG)
		_ = version1 // ignore whatever version they propose
	}
	if err == nil {
		// DEBUG
		fmt.Println("server has decoded hello")
		// END
		version2 := serverVersion
		sSession, ciphertextOut, err := xa.ServerEncryptHelloReply(
			sOneShot, uint32(version2))
		if err == nil {
			// we have our AesSession
			h.AesSession = *sSession
			// The server has preceded the ciphertext with the plain text IV.
			err = h.WriteData(ciphertextOut)
		}
		if err == nil {
			h.version = uint32(version2)
			h.State = HELLO_RCVD
		}
	}
	// On any error silently close the connection.
	if err != nil {
		// DEBUG
		fmt.Printf("handleHello closing cnx, error was %v\n", err)
		// END
		h.Cnx.Close()
	}
	// DEBUG
	if err == nil {
		fmt.Println("  leaving handleHello with no error")
	}
	// END
	return
}
