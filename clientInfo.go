package reg

// xlReg_go/client_info.go

// This file contains functions and structures used to describe
// and manage the cluster data managed by the registry.

import (
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	xc "github.com/jddixon/xlCrypto_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	"strings"
)

var _ = fmt.Print

type ClientInfo struct {
	Attrs       uint64   //  bit flags are defined in const.go
	MyEnds      []string // serialized EndPointI
	xn.BaseNode          // name and ID must be unique
}

func NewClientInfo(name string, id *xi.NodeID,
	commsPubKey, sigPubKey *rsa.PublicKey, attrs uint64, myEnds []string) (
	client *ClientInfo, err error) {

	// all attrs bits are zero by default

	// DEBUG
	// fmt.Printf("NewClientInfo for server %s\n", name)
	// END
	base, err := xn.NewBaseNode(name, id, commsPubKey, sigPubKey, nil)
	if err == nil {
		client = &ClientInfo{
			Attrs:    attrs,
			MyEnds:   myEnds,
			BaseNode: *base,
		}
	}
	return
}

// Create the ClientInfo corresponding to the token passed.

func NewClientInfoFromToken(token *XLRegMsg_Token) (
	m *ClientInfo, err error) {

	var nodeID *xi.NodeID
	if token == nil {
		err = NilToken
	} else {
		nodeID, err = xi.New(token.GetID())
		if err == nil {
			ck, err := xc.RSAPubKeyFromWire(token.GetCommsKey())
			if err == nil {
				sk, err := xc.RSAPubKeyFromWire(token.GetSigKey())
				if err == nil {
					m, err = NewClientInfo(token.GetName(), nodeID,
						ck, sk, token.GetAttrs(), token.GetMyEnds())
				}
			}
		}
	}
	return
}

// Return the XLRegMsg_Token corresponding to this cluster client.
func (mi *ClientInfo) Token() (token *XLRegMsg_Token, err error) {

	var ckBytes, skBytes []byte

	ck := mi.GetCommsPublicKey()
	// DEBUG
	if ck == nil {
		fmt.Printf("ClientInfo.Token: %s commsPubKey is nil\n", mi.GetName())
	}
	// END
	ckBytes, err = xc.RSAPubKeyToWire(ck)
	if err == nil {
		skBytes, err = xc.RSAPubKeyToWire(mi.GetSigPublicKey())
		if err == nil {
			name := mi.GetName()
			token = &XLRegMsg_Token{
				Name:     &name,
				Attrs:    &mi.Attrs,
				ID:       mi.GetNodeID().Value(),
				CommsKey: ckBytes,
				SigKey:   skBytes,
				MyEnds:   mi.MyEnds,
			}
		}
	}
	return
}

// EQUAL ////////////////////////////////////////////////////////////

func (mi *ClientInfo) Equal(any interface{}) bool {

	if any == mi {
		return true
	}
	if any == nil {
		return false
	}
	switch v := any.(type) {
	case *ClientInfo:
		_ = v
	default:
		return false
	}
	other := any.(*ClientInfo) // type assertion
	if mi.Attrs != other.Attrs {
		return false
	}
	if mi.MyEnds == nil {
		if other.MyEnds != nil {
			return false
		}
	} else {
		if other.MyEnds == nil {
			return false
		}
		if len(mi.MyEnds) != len(other.MyEnds) {
			return false
		}
		for i := 0; i < len(mi.MyEnds); i++ {
			if mi.MyEnds[i] != other.MyEnds[i] {
				return false
			}
		}
	}
	// WARNING: panics without the ampersand !
	return mi.BaseNode.Equal(&other.BaseNode)
}

// SERIALIZATION ////////////////////////////////////////////////////

func (mi *ClientInfo) Strings() (ss []string) {
	ss = []string{"clientInfo {"}
	bns := mi.BaseNode.Strings()
	for i := 0; i < len(bns); i++ {
		ss = append(ss, "    "+bns[i])
	}
	ss = append(ss, fmt.Sprintf("    attrs: 0x%016x", mi.Attrs))
	ss = append(ss, "    endPoints {")
	for i := 0; i < len(mi.MyEnds); i++ {
		ss = append(ss, "        "+mi.MyEnds[i])
	}
	ss = append(ss, "    }")
	ss = append(ss, "}")
	return
}

func (mi *ClientInfo) String() string {
	return strings.Join(mi.Strings(), "\n")
}
func collectClientAttrs(mi *ClientInfo, ss []string) (rest []string, err error) {
	rest = ss
	var line string
	line, err = xn.NextNBLine(&rest) // trims
	if err == nil {
		// attrs line looks like "attrs: 0xHHHH..." where H is a hex digit
		if strings.HasPrefix(line, "attrs: 0x") {
			var val []byte
			var attrs uint64
			line := line[9:]
			val, err = hex.DecodeString(line)
			if err == nil {
				if len(val) != 8 {
					err = WrongNumberOfBytesInAttrs
				} else {
					for i := 0; i < 8; i++ {
						// assume little-endian ; but printf has put
						// high order bytes first - ie, it's big-endian
						attrs |= uint64(val[i]) << uint(8*(7-i))
					}
					mi.Attrs = attrs
				}
			}
		} else {
			err = BadAttrsLine
		}
	}
	return
}
func collectMyEnds(mi *ClientInfo, ss []string) (rest []string, err error) {
	rest = ss
	var line string
	line, err = xn.NextNBLine(&rest)
	if err == nil {
		if line == "endPoints {" {
			for {
				line = strings.TrimSpace(rest[0]) // peek
				if line == "}" {
					break
				}
				line, err = xn.NextNBLine(&rest)
				if err == nil {
					// XXX NO CHECK THAT THIS IS A VALID ENDPOINT
					mi.MyEnds = append(mi.MyEnds, line)
				}
			}
			if err == nil {
				line, err = xn.NextNBLine(&rest)
				if err == nil {
					if line != "}" {
						err = MissingClosingBrace
					}
				}
			}
		} else {
			err = MissingEndPointsSection
		}
	}
	return
}
func ParseClientInfo(s string) (
	mi *ClientInfo, rest []string, err error) {

	ss := strings.Split(s, "\n")
	return ParseClientInfoFromStrings(ss)
}

func ParseClientInfoFromStrings(ss []string) (
	mi *ClientInfo, rest []string, err error) {

	bn, rest, err := xn.ParseBNFromStrings(ss, "clientInfo")
	if err == nil {
		mi = &ClientInfo{BaseNode: *bn}
		rest, err = collectClientAttrs(mi, rest)
		if err == nil {
			rest, err = collectMyEnds(mi, rest)
		}
		if err == nil {
			var line string
			// expect and consume a closing brace
			line, err = xn.NextNBLine(&rest)
			if err == nil {
				if line != "}" {
					err = MissingClosingBrace
				}
			}
		}
	}
	return
}
