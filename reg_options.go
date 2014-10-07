package reg

// xlReg_go/reg_options.go

import (
	xt "github.com/jddixon/xlTransport_go"
	"log"
)

// Options normally set from the command line or derived from those.
// Not used in this package but used by xlReg
type RegOptions struct {
	Address     string
	BackingFile string
	ClearFilter bool
	EndPoint    xt.EndPointI // derived from Address, Port
	Ephemeral   bool         // XXX probably don't need
	K           uint
	Lfs         string
	Logger      *log.Logger
	M           uint
	Name        string
	Port        string
	Testing     bool
	T           uint
	Verbose     bool
}
