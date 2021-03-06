package reg

// xlReg_go/registry.go

// This file contains functions and structures used to describe
// and manage the cluster data managed by the registry.

import (
	"fmt"
	ha "github.com/jddixon/hamt_go"
	xcl "github.com/jddixon/xlCluster_go"
	xf "github.com/jddixon/xlCrypto_go/filters"
	xi "github.com/jddixon/xlNodeID_go"
	xt "github.com/jddixon/xlTransport_go"
	xu "github.com/jddixon/xlUtil_go"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var _ = fmt.Print

type Registry struct {
	LogFile string
	Logger  *log.Logger // volatile, not serialized

	// registry data
	m, k           uint          // serialized
	Clusters       []*RegCluster // serialized
	GlobalEndPoint xt.EndPointI  // gets written to regCred
	idFilter       xf.BloomSHAI

	ClustersByName map[string]*RegCluster // volatile, not serialized
	ClustersByID   ha.HAMT                // -ditto-
	RegMembersByID ha.HAMT                // -ditto-
	mu             sync.RWMutex           // -ditto-

	// the extended XLattice node, so id, lfs, keys, etc
	RegNode
}

func NewRegistry(clusters []*RegCluster,
	rn *RegNode,
	opt *RegOptions) (
	reg *Registry, err error) {

	var (
		idFilter      xf.BloomSHAI
		m             ha.HAMT
		serverVersion xu.DecimalVersion
	)
	serverVersion, err = xu.ParseDecimalVersion(VERSION)
	if err == nil && rn == nil {
		err = NilRegNode
	}
	if err == nil {
		if opt.BackingFile == "" {
			idFilter, err = xf.NewBloomSHA(opt.M, opt.K)
		} else {
			idFilter, err = xf.NewMappedBloomSHA(opt.M, opt.K, opt.BackingFile)
		}
		if err == nil {
			// HAMT root table size is 2^opt.T
			m, err = ha.NewHAMT(DEFAULT_W, opt.T)
		}
	}
	if err == nil {
		logger := opt.Logger
		if logger == nil {
			logger = log.New(os.Stderr, "", log.Ldate|log.Ltime)
		}
		testing := opt.Testing
		// DEBUG
		if testing {
			logger.Printf("testing and globalEndPoint = %s\n", opt.GlobalEndPoint)
		}
		// END
		reg = &Registry{
			idFilter:       idFilter,
			Clusters:       clusters,
			GlobalEndPoint: opt.GlobalEndPoint, // written to regCred
			ClustersByName: make(map[string]*RegCluster),
			ClustersByID:   m,

			Logger:  logger,
			RegNode: *rn,
		}
		if clusters != nil {
			// XXX need to populate the indexes here
		}
		myLFS := rn.GetLFS()
		if myLFS != "" {
			var ep []xt.EndPointI
			// list of endpoints for regCred
			//for i := 0; i < rn.SizeEndPoints(); i++ {
			//	ep = append(ep, rn.GetEndPoint(i))
			//}
			// XXX err should never be nil here
			if err == nil {
				ep = append(ep, opt.GlobalEndPoint)
				regCred := &RegCred{
					Name:        rn.GetName(),
					ID:          rn.GetNodeID(),
					CommsPubKey: rn.GetCommsPublicKey(),
					SigPubKey:   rn.GetSigPublicKey(),
					EndPoints:   ep,
					Version:     serverVersion,
				}
				serialized := regCred.String() // shd have terminating CRLF
				logger.Print(serialized)
				pathToFile := filepath.Join(myLFS, "regCred.dat")
				err = ioutil.WriteFile(pathToFile, []byte(serialized), 0644)
			}
		}
	}
	if err == nil {
		// registry's own ID added to Bloom filter
		regID := rn.GetNodeID()
		// DEBUG
		reg.Logger.Printf("\ninserting registry ID %x\n", regID.Value())
		// END
		err = reg.InsertID(regID)
	}
	return
}

func (reg *Registry) ContainsID(n *xi.NodeID) (found bool, err error) {
	found, _, err = reg.idFilter.IsMember(n.Value())

	// DEBUG
	reg.Logger.Printf("Registry.ContainsID(%x) returning %v, %v\n",
		n.Value(), found, err)
	// END
	return
}
func (reg *Registry) InsertID(n *xi.NodeID) (err error) {
	b := n.Value()
	// DEBUG
	reg.Logger.Printf("Registry.InsertID(%x)\n", b)
	// END
	found, _, err := reg.idFilter.IsMember(b)
	if err == nil {
		if found {
			// DEBUG
			reg.Logger.Printf("  id is already registered\n")
			// END
			err = IDAlreadyInUse
		} else {
			err = reg.idFilter.Insert(b)
			// DEBUG
			reg.Logger.Printf("  added %x to registry; err is %v\n", b, err)
			// END
		}
	}
	return
}
func (reg *Registry) IDCount() (count uint) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	count = reg.idFilter.Size()
	return
}

// XXX RegMembersByID is not being updated!  This is the redundant and so
// possibly inconsistent index of members of registry clusters

// Add a cluster member to the RegMembersByID index unless it is already
// present.
func (reg *Registry) indexClusterMember(cluster *RegCluster, cm *xcl.ClusterMember) (
	index int, err error) {

	if cluster == nil {
		err = NilCluster
	} else if cm == nil {
		err = NilClusterMember
	} else {

	}
	return
}
func (reg *Registry) AddCluster(cluster *RegCluster) (index int, err error) {

	var bKey, cKey ha.BytesKey
	if cluster == nil {
		err = NilCluster
	} else {
		name := cluster.Name
		id := cluster.ID // []byte

		// convert ID into a HAMT BytesKey
		bKey, err = ha.NewBytesKey(id)
		if err == nil {
			reg.mu.Lock()
			defer reg.mu.Unlock()

			if _, ok := reg.ClustersByName[name]; ok {
				err = NameAlreadyInUse
			} else {
				var whatever interface{}
				whatever, err = reg.ClustersByID.Find(bKey)
				if err == nil && whatever != nil {
					err = IDAlreadyInUse
				}
			}
			if err == nil {
				index = len(reg.Clusters)
				reg.Clusters = append(reg.Clusters, cluster)
				reg.ClustersByName[name] = cluster
				cKey, err = ha.NewBytesKey(cluster.GetNodeID().Value())
				if err == nil {
					err = reg.ClustersByID.Insert(cKey, cluster)
				}
			}
		}
	}
	if err != nil {
		index = -1
	}
	return
}

// This function generates a good-quality random NodeID (a 20-byte
// value) that is not already known to the registry and then adds
// the new NodeID to the registry's Bloom filter.
func (reg *Registry) InsertUniqueNodeID() (nodeID *xi.NodeID, err error) {

	nodeID, err = xi.New(nil)
	found, err := reg.ContainsID(nodeID)
	for err == nil && found {
		nodeID, err = xi.New(nil)
		found, err = reg.ContainsID(nodeID)
	}
	if err == nil {
		err = reg.idFilter.Insert(nodeID.Value())
	}
	// DEBUG
	reg.Logger.Printf("InsertUniqueNodeID returning %x, %v\n",
		nodeID.Value(), err)
	// END
	return
}

// SERIALIZATION ====================================================

// Tentatively the registry is serialized separately from the regNode
// and so consists of a sequence of serialized clusters

func (reg *Registry) String() (s string) {
	return strings.Join(reg.Strings(), "\n")
}

// If we change the serialization so that there is no closing brace,
// it will be possible to simply append cluster serializations to the
// registry configuration file while the registry is running.

func (reg *Registry) Strings() (ss []string) {
	ss = []string{"registry {"}
	ss = append(ss, fmt.Sprintf("    LogFile: %s", reg.LogFile))
	ss = append(ss, "}")

	for i := 0; i < len(reg.Clusters); i++ {
		cs := reg.Clusters[i].Strings()
		for j := 0; j < len(cs); j++ {
			ss = append(ss, cs[j])
		}
	}
	return
}

func ParseRegistry(s string) (reg *Registry, rest []string, err error) {

	// XXX STUB
	return
}
