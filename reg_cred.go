package reg

// xlReg_go/reg_cred.go

import (
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	xc "github.com/jddixon/xlCrypto_go"
	xi "github.com/jddixon/xlNodeID_go"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
	"strings"
)

var _ = fmt.Print

type RegCred struct {
	Name        string
	ID          *xi.NodeID
	CommsPubKey *rsa.PublicKey    // CommsKey in token
	SigPubKey   *rsa.PublicKey    // SigKey in token
	EndPoints   []xt.EndPointI    // MyEnds in token
	Version     xu.DecimalVersion // not in token at all
}

func (rc *RegCred) String() string {

	// Ignore possible errors; the values in this data structure
	// should have already been checked.
	ck_, _ := xc.RSAPubKeyToDisk(rc.CommsPubKey)
	sk_, _ := xc.RSAPubKeyToDisk(rc.SigPubKey)
	ck := string(ck_)
	sk := string(sk_)

	// XXX Space after the colon is strictly required
	ss := []string{"regCred {"}
	ss = append(ss, "    Name: "+rc.Name)
	ss = append(ss, "    ID: "+rc.ID.String())
	ss = append(ss, "    CommsPubKey: "+ck)
	ss = append(ss, "    SigPubKey: "+sk)
	ss = append(ss, "    EndPoints {")

	for i := 0; i < len(rc.EndPoints); i++ {
		ss = append(ss, "         "+rc.EndPoints[i].String())
	}
	ss = append(ss, "    }")
	ss = append(ss, "    Version: "+rc.Version.String())
	ss = append(ss, "}")
	return strings.Join(ss, "\r\n") + "\r\n"
}

func ParseRegCred(s string) (rc *RegCred, err error) {

	var (
		line    string
		parts   []string
		name    string
		nodeID  *xi.NodeID
		ck, sk  *rsa.PublicKey
		e       []xt.EndPointI
		version xu.DecimalVersion
	)
	ss := strings.Split(s, "\n")
	line, err = xc.NextNBLine(&ss)
	if (err == nil) && (line != "regCred {") {
		err = IllFormedRegCred
	}
	if err == nil {
		line, err = xc.NextNBLine(&ss)
		if err == nil {
			parts = strings.Split(line, ": ")
			if len(parts) == 2 && parts[0] == "Name" {
				name = strings.TrimLeft(parts[1], " \t")
			} else {
				err = IllFormedRegCred
			}
		}
	}
	if err == nil {
		var id []byte
		line, err = xc.NextNBLine(&ss)
		if err == nil {
			parts = strings.Split(line, ": ")
			if len(parts) == 2 && parts[0] == "ID" {
				id, err = hex.DecodeString(parts[1])
			} else {
				err = IllFormedRegCred
			}
		}
		if err == nil {
			nodeID, err = xi.New(id)
		}
	}
	if err == nil {
		line, err = xc.NextNBLine(&ss)
		if err == nil {
			parts = strings.Split(line, ": ")
			if len(parts) == 2 && parts[0] == "CommsPubKey" {
				ck, err = xc.RSAPubKeyFromDisk([]byte(parts[1]))
			} else {
				err = IllFormedRegCred
			}
		}
	}
	if err == nil {
		line, err = xc.NextNBLine(&ss)
		if err == nil {
			parts = strings.Split(line, ": ")
			if len(parts) == 2 && parts[0] == "SigPubKey" {
				sk, err = xc.RSAPubKeyFromDisk([]byte(parts[1]))
			} else {
				err = IllFormedRegCred
			}
		}
	}
	if err == nil {
		line, err = xc.NextNBLine(&ss)
		if err == nil {
			// collect EndPoints section; this should be turned into a
			// utility function
			if line == "EndPoints {" {
				for err == nil {
					line = strings.TrimSpace(ss[0]) // peek
					if line == "}" {
						break
					}
					line, err = xc.NextNBLine(&ss)
					if err == nil {
						line = strings.TrimSpace(line)
						parts := strings.Split(line, ": ")
						if len(parts) != 2 || parts[0] != "TcpEndPoint" {
							err = IllFormedRegCred
						} else {
							var ep xt.EndPointI
							ep, err = xt.NewTcpEndPoint(parts[1])
							if err == nil {
								e = append(e, ep)
							}
						}
					}
				}
				if err == nil {
					line, err = xc.NextNBLine(&ss)
					if (err == nil) && (line != "}") {
						err = MissingClosingBrace
					}
				}
			} else {
				err = MissingEndPointsSection
			}
		}
	}
	if err == nil {
		line, err = xc.NextNBLine(&ss)
		if err == nil {
			parts = strings.Split(line, ": ")
			if len(parts) == 2 && parts[0] == "Version" {
				version, err = xu.ParseDecimalVersion(parts[1])
			} else {
				err = IllFormedRegCred
			}
		}
	}
	if err == nil {
		rc = &RegCred{
			Name:        name,
			ID:          nodeID,
			CommsPubKey: ck,
			SigPubKey:   sk,
			EndPoints:   e,
			Version:     version,
		}
	}
	return
}
