package reg

// xlReg_go/const.go

import (
	xh "github.com/jddixon/hamt_go"
)

const (
	BLOCK_SIZE = 4096

	DEFAULT_M = uint(20)
	DEFAULT_K = uint(8)
	DEFAULT_W = xh.MAX_W // default HAMT w parameter

	MAX_CLUSTER_SIZE = uint32(64) // inclusive
	MIN_CLUSTER_SIZE = uint32(2)
)

// client attrs bits, also used for member attrs
const (
	ATTR_EPHEMERAL = 1 << iota
	ATTR_ADMIN
	ATTR_SOLO // no related cluster, persists config to LFS
)
