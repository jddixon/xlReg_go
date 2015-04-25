package reg

// xlReg_go/cnxHandler.go

import (
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xa "github.com/jddixon/xlProtocol_go/aes_cnx"
	xt "github.com/jddixon/xlTransport_go"
)

var _ = fmt.Print

const (
	MSG_BUF_LEN = 16 * 1024
)

type CnxHandler struct {
	Cnx *xt.TcpConnection
	xa.AesSession
}

// Create a new handler for a connection.  If the random number generator
// supplied is nil, a crypto-quality rng is created for use with this
// connection.  If the key supplied is nil, uses the RNG to create a
// 256-bit AES key.
//
func NewCnxHandler(cnx *xt.TcpConnection, key []byte, rng *xr.PRNG) (
	handler *CnxHandler, err error) {

	if cnx == nil {
		err = BadOrNilCnx
	} else {
		session, err := xa.NewAesSession(key, rng)
		if err == nil {
			handler = &CnxHandler{
				Cnx:        cnx,
				AesSession: *session,
			}
		}
	}
	return
}

// Read data from the connection.  XXX This will not handle partial
// reads correctly.
//
func (a *CnxHandler) ReadData() (data []byte, err error) {
	data = make([]byte, MSG_BUF_LEN)
	count, err := a.Cnx.Read(data)
	if err == nil && count > 0 {
		data = data[:count]
	} else {
		data = nil
	}
	return
}

func (a *CnxHandler) WriteData(data []byte) (err error) {
	count, err := a.Cnx.Write(data)

	// XXX handle cases where not all bytes written

	_ = count
	return
}
func (a *CnxHandler) DecodePacket(data []byte) (*XLRegMsg, error) {
	var m XLRegMsg
	err := proto.Unmarshal(data, &m)
	// XXX do some filtering, eg for nil op
	return &m, err
}

func (a *CnxHandler) EncodePacket(msg *XLRegMsg) (data []byte, err error) {
	return proto.Marshal(msg)
}

func (a *CnxHandler) EncodePadEncrypt(msg *XLRegMsg) (
	pCiphertext []byte, err error) {

	var iv []byte // superfluous

	cData, err := a.EncodePacket(msg) // serialize the message
	if err == nil {
		// pCiphertext contains the IV followed by ciphertext
		pCiphertext, iv, err = a.Encrypt(cData)
		_ = iv
	}
	return
}

func (a *CnxHandler) DecryptUnpadDecode(pCiphertext []byte) (
	msg *XLRegMsg, err error) {

	plaintext, iv, err := a.Decrypt(pCiphertext)
	_ = iv
	if err == nil {
		msg, err = a.DecodePacket(plaintext)
	}
	return
}
