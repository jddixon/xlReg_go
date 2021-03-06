package reg

// xlReg_go/reg_server.go

import (
	"fmt"
	xt "github.com/jddixon/xlTransport_go"
	"io"
	"net"
)

var _ = fmt.Printf

type RegServer struct {
	Testing   bool // serialized
	Verbosity int  // serialized
	DoneCh    chan (bool)
	Registry
}

func NewRegServer(reg *Registry, testing bool, verbosity int) (
	rs *RegServer, err error) {

	if reg == nil {
		err = NilRegistry
	} else {
		rs = &RegServer{
			Testing:   testing,
			Verbosity: verbosity,
			Registry:  *reg,
			DoneCh:    make(chan bool, 1),
		}
	}
	return
}

func (rs *RegServer) Stop() {
	acc := rs.GetAcceptor()
	if acc != nil {
		acc.Close()
	}
}
func (rs *RegServer) GetAcceptor() xt.AcceptorI {
	return rs.Registry.GetAcceptor(0)
}

// Starts the server running in a goroutine.  Does not block.
func (rs *RegServer) Start() (err error) {

	err = rs.OpenAcc() // opens RegNode's acceptor
	// DEBUG
	acc := rs.GetAcceptor()
	if acc == nil {
		fmt.Printf("RegServer.Start: acceptor is NIL\n")
	} else {
		fmt.Printf("RegServer.Start(): acceptor is %s\n", acc.String())
	}
	// END
	if err == nil {
		go func() {
			for {
				logger := rs.Registry.Logger

				// As each client connects its connection is passed to a
				// handler running in a separate goroutine.
				cnx, err := rs.GetAcceptor().Accept()
				if err != nil {
					// SHOULD NOT CONTINUE IF 'use of closed network connection";
					// this yields an infinite loop if the listening socket has
					// been closed to shut down the server.
					netOpError, ok := err.(*net.OpError)
					if ok && netOpError.Err.Error() == "use of closed network connection" {
						err = nil
					} else {
						logger.Printf(
							"fatal I/O error %v, shutting down the server\n",
							err)
					}
					break
				}
				go func() {
					var (
						h *InHandler
					)
					h, err = NewInHandler(&rs.Registry, cnx)
					if err == nil {
						err = h.Start()
					}
					if err != nil {
						if err != io.EOF {
							logger.Printf(
								"I/O error %v, closing client connection\n", err)
						}
						cnx := h.Cnx
						if cnx != nil {
							cnx.Close()
						}
					}
				}()
			}
			rs.DoneCh <- true
		}()
	}
	return
}

// SERIALIZATION ====================================================

func ParseRegServer(s string) (rs *RegServer, rest []string, err error) {

	// XXX STUB
	return
}

func (rs *RegServer) String() (s string) {

	// STUB XXX
	return
}

func (rs *RegServer) Strings() (s []string) {

	// STUB XXX
	return
}
