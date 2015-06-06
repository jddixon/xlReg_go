package reg

// xlReg_go/reg_node.go

// We collect functions and structures relating to the operation
// of the registry as a communicating server here.

import (
	"crypto/rsa"
	"fmt"
	xc "github.com/jddixon/xlCrypto_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	xm "github.com/jddixon/xlattice_go/msg"
	"strings"
)

var _ = fmt.Print

type RegNode struct {
	Acc       xt.AcceptorI    // a convenience here, so not serialized
	StopCh    chan bool       // volatile, so not serialized
	StoppedCh chan bool       // -ditto-
	ckPriv    *rsa.PrivateKey // duplicate to allow simple
	skPriv    *rsa.PrivateKey // access in this package
	xn.Node                   // name, id, ck, sk, etc, etc
}

/**
 * Returns AcceptorNotLive if the node does not have an open acceptor.
 */
func NewRegNode(node *xn.Node, commsKey, sigKey *rsa.PrivateKey) (
	q *RegNode, err error) {

	var acc xt.AcceptorI

	if node == nil {
		err = NilNode
		// We would prefer that the node's name be xlReg and that its
		// LFS default to /var/app/xlReg.
	} else {
		acc = node.GetAcceptor(0)
		if acc == nil {
			err = xm.AcceptorNotLive
		}
		if err == nil {
			stopCh := make(chan bool, 1)
			stoppedCh := make(chan bool, 1)

			q = &RegNode{
				Acc:       acc,
				StopCh:    stopCh,
				StoppedCh: stoppedCh,
				ckPriv:    commsKey,
				skPriv:    sigKey,
				Node:      *node,
			}
		}
	}
	return
}

// SERIALIZATION ====================================================

// A regNode is serialized in more or less reverse order.  The
// "regNode {" line, which is followed by a "node {" line, which
// is followed by the body of the BaseNode, after which comes the
// body of the Node, which is followed by the RegNodes two private keys.

// THE CURRENT IMPLEMENTATION serializes clusters including their
func (rn *RegNode) Strings() (ss []string) {
	ss = []string{"regNode {"}
	ns := rn.Node.Strings()
	for i := 0; i < len(ns); i++ {
		ss = append(ss, "    "+ns[i])
	}
	// XXX Possibly foolish assumption that serialization must succeed.
	ckSer, _ := xc.RSAPrivateKeyToDisk(rn.ckPriv)
	skSer, _ := xc.RSAPrivateKeyToDisk(rn.skPriv)
	ss = append(ss, fmt.Sprintf("    ckPriv: %s", ckSer))
	ss = append(ss, fmt.Sprintf("    skPriv: %s", skSer))
	ss = append(ss, "}")
	return
}

func (rn *RegNode) String() (s string) {
	return strings.Join(rn.Strings(), "\n")
}

func ParseRegNode(s string) (rn *RegNode, rest []string, err error) {
	ss := strings.Split(s, "\n")
	return ParseRegNodeFromStrings(ss)
}

/**
 * Parse a serialized RegNode, returning one with an open acceptor.
 */
func ParseRegNodeFromStrings(ss []string) (
	rn *RegNode, rest []string, err error) {

	var (
		line   string
		node   *xn.Node
		ckPriv *rsa.PrivateKey
		skPriv *rsa.PrivateKey
	)
	rest = ss
	line, err = xn.NextNBLine(&rest)
	if (err == nil) && (line != "regNode {") {
		err = MissingRegNodeLine
	} else {
		node, rest, err = xn.ParseFromStrings(rest)
		if err == nil {
			line, err = xn.NextNBLine(&rest)
			if err == nil {
				parts := strings.Split(line, ": ")
				if parts[0] == "ckPriv" && parts[1] == "-----BEGIN -----" {
					ckPriv, err = xn.ExpectRSAPrivateKey(&rest)
				} else {
					err = MissingPrivateKey
				}
				if err == nil {
					line, err = xn.NextNBLine(&rest)
					if err == nil {
						parts := strings.Split(line, ": ")
						if parts[0] == "skPriv" && parts[1] == "-----BEGIN -----" {
							skPriv, err = xn.ExpectRSAPrivateKey(&rest)
						} else {
							err = MissingPrivateKey
						}
						if err == nil {
							line, err = xn.NextNBLine(&rest)
							if (err == nil) && (line != "}") {
								err = MissingClosingBrace
							}
							if err == nil {
								// Try to open the acceptor.
								err = node.OpenAcc()
								if err == nil {
									rn, err = NewRegNode(node, ckPriv, skPriv)
								}
							}
						}
					}
				}
			}
		}
	}
	return
}
