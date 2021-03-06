package reg

// xlReg_go/reg_cluster.go

// This file contains functions and structures used to describe
// and manage the clusters managed by the registry.

import (
	"bytes"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	ha "github.com/jddixon/hamt_go"
	xc "github.com/jddixon/xlCrypto_go"
	xi "github.com/jddixon/xlNodeID_go"
	xo "github.com/jddixon/xlOverlay_go"
	xm "github.com/jddixon/xlUtil_go/math"
	"strconv"
	"strings"
	"sync"
)

var _ = fmt.Print

// cluster bit flags (Attrs)
const (
	CLUSTER_EPHEMERAL = 1 << iota
	CLUSTER_DELETED
)

type RegCluster struct {
	Name          string // must be globally unique, unique within the registry
	ID            []byte // must be globally unique
	Attrs         uint64 // a field of bit flags
	maxSize       uint32 // a maximum; must be > 0
	epCount       uint32 // a positive integer, for now is 1 or 2
	Members       []*ClientInfo
	MembersByName map[string]*ClientInfo
	MembersByID   ha.HAMT
	mu            sync.RWMutex
}

func NewRegCluster(name string, id *xi.NodeID, attrs uint64,
	maxSize, epCount uint32) (rc *RegCluster, err error) {

	var m ha.HAMT

	if name == "" {
		name = "xlCluster"
	}
	nameMap := make(map[string]*ClientInfo)
	if epCount < 1 {
		err = ClusterMembersMustHaveEndPoint
	}
	if err == nil && maxSize < 1 {
		//err = ClusterMustHaveTwo
		err = ClusterMustHaveMember
	} else {
		t := uint(xm.NextExp2_32(maxSize))
		m, err = ha.NewHAMT(DEFAULT_W, t)
	}
	if err == nil {
		rc = &RegCluster{
			Attrs:         attrs,
			Name:          name,
			ID:            id.Value(),
			epCount:       epCount,
			maxSize:       maxSize,
			MembersByName: nameMap,
			MembersByID:   m,
		}
	}
	return
}

func (rc *RegCluster) AddToCluster(name string, id *xi.NodeID,
	commsPubKey, sigPubKey *rsa.PublicKey, attrs uint64, myEnds []string) (
	err error) {

	member, err := NewClientInfo(
		name, id, commsPubKey, sigPubKey, attrs, myEnds)
	if err == nil {
		err = rc.AddMember(member)
	}
	return
}

func (rc *RegCluster) AddMember(member *ClientInfo) (err error) {

	// verify no existing member has the same name
	name := member.GetName()

	rc.mu.RLock() // <------------------------------------
	_, ok := rc.MembersByName[name]
	rc.mu.RUnlock() // <------------------------------------

	if ok {
		// DEBUG
		fmt.Printf("AddMember: ATTEMPT TO ADD EXISTING MEMBER %s\n", name)
		// END
		err = ClusterMemberNameInUse
	}
	if err == nil {
		var (
			entry interface{}
			bKey  ha.BytesKey
		)
		// check for entry in HAMT
		rc.mu.RLock() // <---------------------------------
		bKey, err = ha.NewBytesKey(rc.ID)
		entry, err = rc.MembersByID.Find(bKey)
		rc.mu.RUnlock() // <-------------------------------
		if err == nil {
			if entry != nil {
				err = ClusterMemberIDInUse
			}
		}
		if err == nil {
			rc.mu.Lock()             // <------------------
			index := len(rc.Members) // DEBUG
			_ = index                // we might want to use this
			rc.Members = append(rc.Members, member)
			rc.MembersByName[name] = member
			bKey, err = ha.NewBytesKey(member.GetNodeID().Value())
			if err == nil {
				err = rc.MembersByID.Insert(bKey, member)
			}
			rc.mu.Unlock() // <----------------------------
		}
	}
	return
}

func (rc *RegCluster) EndPointCount() uint32 {
	return rc.epCount
}
func (rc *RegCluster) MaxSize() uint32 {
	return rc.maxSize
}
func (rc *RegCluster) Size() uint32 {
	var curSize uint32
	rc.mu.RLock() // <-------------------------------------
	curSize = uint32(len(rc.Members))
	rc.mu.RUnlock() // <-----------------------------------
	return curSize
}

// EQUAL ////////////////////////////////////////////////////////////
func (rc *RegCluster) Equal(any interface{}) bool {

	if any == rc {
		return true
	}
	if any == nil {
		return false
	}
	switch v := any.(type) {
	case *RegCluster:
		_ = v
	default:
		return false
	}
	other := any.(*RegCluster) // type assertion
	if rc.Attrs != other.Attrs {
		// DEBUG
		fmt.Printf("rc.Equal: ATTRS DIFFER %s vs %s\n", rc.Attrs, other.Attrs)
		// END
		return false
	}
	if rc.Name != other.Name {
		// DEBUG
		fmt.Printf("rc.Equal: NAMES DIFFER %s vs %s\n", rc.Name, other.Name)
		// END
		return false
	}
	if !bytes.Equal(rc.ID, other.ID) {
		// DEBUG
		rcHexID := hex.EncodeToString(rc.ID)
		otherHexID := hex.EncodeToString(other.ID)
		fmt.Printf("rc.Equal: IDs DIFFER %s vs %s\n", rcHexID, otherHexID)
		// END
		return false
	}
	if rc.epCount != other.epCount {
		// DEBUG
		fmt.Printf("rc.Equal: EPCOUNTS DIFFER %d vs %d\n",
			rc.epCount, other.epCount)
		// END
		return false
	}
	if rc.maxSize != other.maxSize {
		// DEBUG
		fmt.Printf("rc.Equal: MAXSIZES DIFFER %d vs %d\n",
			rc.maxSize, other.maxSize)
		// END
		return false
	}
	if rc.Size() != other.Size() {
		// DEBUG
		fmt.Printf("rc.Equal:ACTUAL SIZES DIFFER %d vs %d\n",
			rc.Size(), other.Size())
		// END
		return false
	}
	// Members			[]*ClientInfo
	for i := uint32(0); i < rc.Size(); i++ {
		rcMember := rc.Members[i]
		otherMember := other.Members[i]
		if !rcMember.Equal(otherMember) {
			return false
		}
	}
	return true
}

// SERIALIZATION ////////////////////////////////////////////////////

func (rc *RegCluster) Strings() (ss []string) {

	ss = []string{"regCluster {"}

	ss = append(ss, fmt.Sprintf("    Attrs: 0x%016x", rc.Attrs))
	ss = append(ss, "    Name: "+rc.Name)
	ss = append(ss, "    ID: "+hex.EncodeToString(rc.ID))
	ss = append(ss, fmt.Sprintf("    epCount: %d", rc.epCount))
	ss = append(ss, fmt.Sprintf("    maxSize: %d", rc.maxSize))

	ss = append(ss, "    Members {")
	for i := 0; i < len(rc.Members); i++ {
		mem := rc.Members[i].Strings()
		for i := 0; i < len(mem); i++ {
			ss = append(ss, "        "+mem[i])
		}
	}
	ss = append(ss, "    }")
	ss = append(ss, "}")

	return
}

func (rc *RegCluster) String() string {
	return strings.Join(rc.Strings(), "\n")
}
func ParseRegCluster(s string) (rc *RegCluster, rest []string, err error) {
	ss := strings.Split(s, "\n")
	return ParseRegClusterFromStrings(ss)
}
func ParseRegClusterFromStrings(ss []string) (
	rc *RegCluster, rest []string, err error) {

	var (
		attrs            uint64
		name             string
		id               *xi.NodeID
		epCount, maxSize uint32
	)
	rest = ss

	var line string
	line, err = xc.NextNBLine(&rest) // the line is trimmed
	if err == nil {
		if line != "regCluster {" {
			fmt.Println("MISSING regCluster {")
			err = IllFormedCluster
		} else {
			line, err = xc.NextNBLine(&rest)
			if err == nil {
				if strings.HasPrefix(line, "Attrs: ") {
					var i int64
					i, err = strconv.ParseInt(line[7:], 0, 64)
					if err == nil {
						attrs = uint64(i)
					}
				} else {
					fmt.Printf("BAD ATTRS in line '%s'", line)
					err = IllFormedCluster
				}
			}
		}
	}
	if err == nil {
		line, err = xc.NextNBLine(&rest)
		if err == nil {
			if strings.HasPrefix(line, "Name: ") {
				name = line[6:]
			} else {
				fmt.Printf("BAD NAME in line '%s'", line)
				err = IllFormedCluster
			}
		}
	}
	if err == nil {
		// collect ID
		line, err = xc.NextNBLine(&rest)
		if err == nil {
			if strings.HasPrefix(line, "ID: ") {
				var val []byte
				val, err = hex.DecodeString(line[4:])
				if err == nil {
					id, err = xi.New(val)
				}
			} else {
				fmt.Println("BAD ID")
				err = IllFormedCluster
			}
		}
	}
	if err == nil {
		line, err = xc.NextNBLine(&rest)
		if err == nil {
			if strings.HasPrefix(line, "epCount: ") {
				var count int
				count, err = strconv.Atoi(line[9:])
				if err == nil {
					epCount = uint32(count)
				}
			} else {
				fmt.Println("BAD END POINT COUNT")
				err = IllFormedCluster
			}
		}
	}
	if err == nil {
		line, err = xc.NextNBLine(&rest)
		if err == nil {
			if strings.HasPrefix(line, "maxSize: ") {
				var size int
				size, err = strconv.Atoi(line[9:])
				if err == nil {
					maxSize = uint32(size)
				}
			} else {
				fmt.Println("BAD MAX_SIZE")
				err = IllFormedCluster
			}
		}
	}
	if err == nil {
		rc, err = NewRegCluster(name, id, attrs, maxSize, epCount)
	}
	if err == nil {
		line, err = xc.NextNBLine(&rest)
		if err == nil {
			if line == "Members {" {
				for {
					line = strings.TrimSpace(rest[0]) // peek
					if line == "}" {
						break
					}
					var member *ClientInfo
					member, rest, err = ParseClientInfoFromStrings(rest)
					if err != nil {
						break
					}
					err = rc.AddMember(member)
					if err != nil {
						break
					}
				}
			} else {
				err = MissingMembersList
			}
		}
	}

	// expect closing brace for Members list
	if err == nil {
		line, err = xc.NextNBLine(&rest)
		if err == nil {
			if line != "}" {
				err = MissingClosingBrace
			}
		}
	}
	// expect closing brace  for cluster
	if err == nil {
		line, err = xc.NextNBLine(&rest)
		if err == nil {
			if line != "}" {
				err = MissingClosingBrace
			}
		}
	}

	return
}

// BaseNodeI INTERFACE //////////////////////////////////////////////

func (rc *RegCluster) GetName() string {
	return rc.Name
}
func (rc *RegCluster) GetNodeID() (id *xi.NodeID) {
	id, _ = xi.New(rc.ID)
	return
}

// Dummy functions to make this compliant with the interface

func (rc *RegCluster) AddOverlay(o xo.OverlayI) (ndx int, err error) {
	return
}
func (rc *RegCluster) SizeOverlays() (size int) {
	return
}
func (rc *RegCluster) GetOverlay(n int) (o xo.OverlayI) {
	return
}

func (rc *RegCluster) GetCommsPublicKey() (ck *rsa.PublicKey) {
	return
}
func (rc *RegCluster) GetSSHCommsPublicKey() (s string) {
	return
}
func (rc *RegCluster) GetSigPublicKey() (sk *rsa.PublicKey) {
	return
}
