package reg

// xlReg_go/member_info.go

// This file contains functions and structures used by registry clients
// to manage information about clusters and their members.

import (
	"crypto/rsa"
	//"encoding/hex"
	"fmt"
	xc "github.com/jddixon/xlCrypto_go"
	xcl "github.com/jddixon/xlCluster_go"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	xt "github.com/jddixon/xlTransport_go"
	//"strings"
)

var _ = fmt.Print

// Create the MemberInfo corresponding to the token passed.

func NewMemberInfoFromToken(token *XLRegMsg_Token) (
	m *xcl.MemberInfo, err error) {

	var (
		ck, sk *rsa.PublicKey
		ctor   xt.ConnectorI
		ctors  []xt.ConnectorI
		farEnd xt.EndPointI
		nodeID *xi.NodeID
		peer   *xn.Peer
	)
	if token == nil {
		err = NilToken
	} else {
		nodeID, err = xi.New(token.GetID())
		if err == nil {
			ck, err = xc.RSAPubKeyFromWire(token.GetCommsKey())
			if err == nil {
				sk, err = xc.RSAPubKeyFromWire(token.GetSigKey())
				if err == nil {
					attrs := token.GetAttrs()
					myEnds := token.GetMyEnds()
					for i := 0; i < len(myEnds); i++ {
						myEnd := myEnds[i]
						farEnd, err = xt.NewTcpEndPoint(myEnd)
						if err != nil {
							break
						}
						ctor, err = xt.NewTcpConnector(farEnd)
						if err != nil {
							break
						}
						ctors = append(ctors, ctor)
					}
					if err == nil {
						peer, err = xn.NewPeer(token.GetName(), nodeID,
							ck, sk, nil, ctors)
						if err == nil {
							m = &xcl.MemberInfo{
								Attrs: attrs,
								Peer:  peer,
							}
						}
					}
				}
				//if err == nil {
				//	m, err = NewMemberInfo(token.GetName(), nodeID,
				//		ck, sk, token.GetAttrs(), token.GetMyEnds())
				//}
			}
		}
	}
	return
}

// Return the XLRegMsg_Token corresponding to this cluster member.
func TokenFromMemberInfo(mi *xcl.MemberInfo) (
	token *XLRegMsg_Token, err error) {

	var ckBytes, skBytes []byte

	ck := mi.Peer.GetCommsPublicKey()
	ckBytes, err = xc.RSAPubKeyToWire(ck)
	if err == nil {
		skBytes, err = xc.RSAPubKeyToWire(mi.Peer.GetSigPublicKey())
		if err == nil {
			name := mi.Peer.GetName()
			var myEnds []string
			for i := 0; i < mi.Peer.SizeConnectors(); i++ {
				myEnds = append(myEnds, mi.Peer.GetConnector(i).String())
			}
			token = &XLRegMsg_Token{
				Name:     &name,
				Attrs:    &mi.Attrs,
				ID:       mi.Peer.GetNodeID().Value(),
				CommsKey: ckBytes,
				SigKey:   skBytes,
				MyEnds:   myEnds,
			}
		}
	}
	return
}
