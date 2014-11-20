package reg

// xlReg_go/reg_cluster_test.go

import (
	"fmt"
	ha "github.com/jddixon/hamt_go"
	xr "github.com/jddixon/rnglib_go"
	. "gopkg.in/check.v1"
)

func (s *XLSuite) TestRegClusterMaker(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_REG_CLUSTER_MAKER")
	}
	var err error
	rng := xr.MakeSimpleRNG()

	// Generate a random cluster
	epCount := uint32(1 + rng.Intn(3)) // so from 1 to 3
	maxSize := uint32(2 + rng.Intn(6)) // so from 2 to 7
	cl := s.makeARegCluster(c, rng, epCount, maxSize)

	c.Assert(cl.MaxSize(), Equals, maxSize)
	c.Assert(cl.Size(), Equals, maxSize) //

	// Verify that member names are unique within the cluster
	ids := make([][]byte, maxSize)
	names := make([]string, maxSize)
	nameMap := make(map[string]uint32)
	for i := uint32(0); i < maxSize; i++ {
		member := cl.Members[i]
		names[i] = member.GetName()
		// fmt.Printf("member[%d]: %s\n", i, names[i])		// DEBUG
		nameMap[names[i]] = i

		// collect IDs while we are at it
		id := member.GetNodeID().Value() // returns a clone of the nodeID
		ids[i] = id
	}
	// if the names are not unique, map will be smaller
	c.Assert(maxSize, Equals, uint32(len(nameMap)))

	// verify that the RegCluster.MembersByName index is correct
	for i := uint32(0); i < maxSize; i++ {
		name := names[i]
		member := cl.MembersByName[name]
		c.Assert(name, Equals, member.GetName())
	}

	// verify that the RegCluster.MembersByID index is correct
	count := uint32(0) // number of successful type assertions
	for i := uint32(0); i < maxSize; i++ {
		id := ids[i]
		var bKey ha.BytesKey
		bKey, err = ha.NewBytesKey(id)
		c.Assert(err, IsNil)
		mbr, err := cl.MembersByID.Find(bKey)
		c.Assert(err, IsNil)
		var member *ClientInfo
		// verify that the type assertion succeeds
		if m, ok := mbr.(*ClientInfo); ok {
			member = m
			mID := member.GetNodeID().Value()
			c.Assert(len(id), Equals, len(mID))
			for j := uint(0); j < uint(len(id)); j++ {
				c.Assert(id[j], Equals, mID[j])
			}
			count++
		}
	}
	c.Assert(maxSize, Equals, count)
}
func (s *XLSuite) TestRegClusterSerialization(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_REG_CLUSTER_SERIALIZATION")
	}
	rng := xr.MakeSimpleRNG()

	// Generate a random cluster
	epCount := uint32(1 + rng.Intn(3)) // so from 1 to 3
	size := uint32(2 + rng.Intn(6))    // so from 2 to 7
	cl := s.makeARegCluster(c, rng, epCount, size)

	// Serialize it
	serialized := cl.String()

	// Reverse the serialization
	deserialized, rest, err := ParseRegCluster(serialized)
	c.Assert(err, IsNil)
	c.Assert(deserialized, Not(IsNil))
	c.Assert(len(rest), Equals, 0)

	// Verify that the deserialized cluster is identical to the original
	c.Assert(deserialized.Equal(cl), Equals, true)

}
