package main

// xlReg_go/cmd/xlReg/xlReg.go

import (
	"crypto/rand"
	"crypto/rsa"
	"flag"
	"fmt"
	xi "github.com/jddixon/xlNodeID_go"
	xn "github.com/jddixon/xlNode_go"
	reg "github.com/jddixon/xlReg_go"
	xt "github.com/jddixon/xlTransport_go"
	xf "github.com/jddixon/xlUtil_go/lfs"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func Usage() {
	fmt.Printf("Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Printf("where the options are:\n")
	flag.PrintDefaults()
}

const (
	DEFAULT_ADDR        = "0.0.0.0" // listen on all interfaces
	DEFAULT_GLOBAL_ADDR = "54.186.197.123"
	DEFAULT_NAME        = "xlReg"
	DEFAULT_LFS         = "/var/app/xlReg"
	TEST_DEFAULT_PORT   = 45678 // for the registry, not clients
	DEFAULT_PORT        = 56789 // for the registry, not clients
)

var (
	// these need to be referenced as pointers
	address = flag.String("a", DEFAULT_ADDR,
		"registry IP address (dotted quad)")
	clearFilter = flag.Bool("c", false,
		"clear Bloom filer at beginning of run")
	ephemeral = flag.Bool("e", false,
		"server is ephemeral, does not persist data")
	globalAddress = flag.String("g", DEFAULT_GLOBAL_ADDR,
		"registry global IP address (dotted quad)")
	justShow = flag.Bool("j", false,
		"display option settings and exit")
	k = flag.Uint("k", reg.DEFAULT_K,
		"number of hash functions in Bloom filter")
	lfs = flag.String("lfs", DEFAULT_LFS,
		"path to work directory")
	logFile = flag.String("l", "log",
		"path to log file")
	m = flag.Uint("m", reg.DEFAULT_M,
		"exponent in Bloom filter")
	name = flag.String("n", DEFAULT_NAME,
		"registry name")
	port = flag.Int("p", DEFAULT_PORT,
		"registry listening port")
	t = flag.Uint("t", reg.DEFAULT_W,
		"number of cells in root HAMT table, exponent of 2")
	testing = flag.Bool("T", false,
		"this is a test run")
	verbose = flag.Bool("v", false,
		"be talkative")
)

func init() {
	fmt.Println("init() invocation") // DEBUG
}

func main() {
	var err error

	flag.Usage = Usage
	flag.Parse()

	// FIXUPS ///////////////////////////////////////////////////////

	if err != nil {
		fmt.Println("error processing NodeID: %s\n", err.Error())
		os.Exit(-1)
	}
	if *testing {
		if *name == DEFAULT_NAME || *name == "" {
			*name = "testReg"
		}
		if *lfs == DEFAULT_LFS || *lfs == "" {
			*lfs = "./myApp/xlReg"
		} else {
			*lfs = path.Join("tmp", *lfs)
		}
		if *address == DEFAULT_ADDR {
			*address = "127.0.0.1"
		}
		if *globalAddress == DEFAULT_GLOBAL_ADDR {
			*globalAddress = "127.0.0.1"
		}
		if *port == DEFAULT_PORT || *port == 0 {
			*port = TEST_DEFAULT_PORT
		}
	}
	var backingFile string
	if !*ephemeral {
		backingFile = path.Join(*lfs, "idFilter.dat")
	}
	addrAndPort := fmt.Sprintf("%s:%d", *address, *port)
	endPoint, err := xt.NewTcpEndPoint(addrAndPort)
	if err != nil {
		fmt.Printf("not a valid endPoint: %s\n", addrAndPort)
		Usage()
		os.Exit(-1)
	}
	globalAddrAndPort := fmt.Sprintf("%s:%d", *globalAddress, *port)
	globalEndPoint, err := xt.NewTcpEndPoint(globalAddrAndPort)
	if err != nil {
		fmt.Printf("not a valid endPoint: %s\n", globalAddrAndPort)
		Usage()
		os.Exit(-1)
	}

	// SANITY CHECKS ////////////////////////////////////////////////
	if err == nil {
		if *m < 2 {
			*m = 20
		}
		if *k < 2 {
			*k = 8
		}
		err = xf.CheckLFS(*lfs, 0700) // tries to create if it doesn't exist
		if err == nil {
			if *logFile != "" {
				*logFile = path.Join(*lfs, *logFile)
			}
		}
	}
	// DISPLAY STUFF ////////////////////////////////////////////////
	if *verbose || *justShow {
		fmt.Printf("address          = %v\n", *address)
		fmt.Printf("backingFile      = %v\n", backingFile)
		fmt.Printf("clearFilter      = %v\n", *clearFilter)
		fmt.Printf("endPoint         = %v\n", endPoint)
		fmt.Printf("ephemeral        = %v\n", *ephemeral)
		fmt.Printf("globalAddress    = %v\n", *globalAddress)
		fmt.Printf("globalEndPoint   = %v\n", *globalEndPoint)
		fmt.Printf("justShow         = %v\n", *justShow)
		fmt.Printf("k                = %d\n", *k)
		fmt.Printf("lfs              = %s\n", *lfs)
		fmt.Printf("logFile          = %s\n", *logFile)
		fmt.Printf("m                = %d\n", *m)
		fmt.Printf("name             = %s\n", *name)
		fmt.Printf("port             = %d\n", *port)
		fmt.Printf("testing          = %v\n", *testing)
		fmt.Printf("verbose          = %v\n", *verbose)
	}
	if *justShow {
		return
	}
	// SET UP OPTIONS ///////////////////////////////////////////////
	var (
		f      *os.File
		logger *log.Logger
		opt    reg.RegOptions
		rs     *reg.RegServer
	)
	if *logFile != "" {
		f, err = os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
		if err == nil {
			logger = log.New(f, "", log.Ldate|log.Ltime)
		}
	}
	if f != nil {
		defer f.Close()
	}
	if err == nil {
		opt.Address = *address
		opt.BackingFile = backingFile
		opt.ClearFilter = *clearFilter
		opt.Ephemeral = *ephemeral
		opt.GlobalEndPoint = globalEndPoint
		opt.K = uint(*k)
		opt.Lfs = *lfs
		opt.Logger = logger
		opt.M = uint(*m)
		opt.Lfs = *lfs
		opt.Port = fmt.Sprintf("%d", *port)
		opt.T = *t
		opt.Testing = *testing
		opt.Verbose = *verbose

		rs, err = setup(&opt)
		if err == nil {
			err = serve(rs)
		}
	}
	_ = logger // NOT YET
	_ = err
}
func setup(opt *reg.RegOptions) (rs *reg.RegServer, err error) {
	// If LFS/.xlattice/reg.config exists, we load that.  Otherwise we
	// create a node.  In either case we force the node to listen on
	// the designated port

	var (
		e                []xt.EndPointI
		node             *xn.Node
		pathToConfigFile string
		rn               *reg.RegNode
		ckPriv, skPriv   *rsa.PrivateKey
	)
	logger := opt.Logger
	verbose := opt.Verbose

	greetings := fmt.Sprintf("xlReg v%s %s start run\n",
		reg.VERSION, reg.VERSION_DATE)
	if verbose {
		fmt.Print(greetings)
	}
	logger.Print(greetings)

	pathToConfigFile = path.Join(path.Join(opt.Lfs, ".xlattice"), "reg.config")
	found, err := xf.PathExists(pathToConfigFile)
	if err == nil {
		if found {
			logger.Printf("Loading existing reg config from %s\n",
				pathToConfigFile)
			// The registry node already exists.  Parse it and we are done.
			var data []byte
			data, err = ioutil.ReadFile(pathToConfigFile)
			if err == nil {
				rn, _, err = reg.ParseRegNode(string(data))
			}
		} else {
			logger.Println("No config file found, creating new registry.")
			// We need to create a registry node from scratch.
			nodeID, _ := xi.New(nil)
			ep, err := xt.NewTcpEndPoint(opt.Address + ":" + opt.Port)
			if err == nil {
				e = []xt.EndPointI{ep}
				ckPriv, err = rsa.GenerateKey(rand.Reader, 2048)
				if err == nil {
					skPriv, err = rsa.GenerateKey(rand.Reader, 2048)
				}
				if err == nil {
					node, err = xn.New("xlReg", nodeID, opt.Lfs, ckPriv, skPriv,
						nil, e, nil)
					if err == nil {
						node.OpenAcc() // XXX needs a complementary close
						if err == nil {
							// DEBUG
							fmt.Printf("XLattice node successfully created\n")
							fmt.Printf("  listening on %s\n", ep.String())
							// END
							rn, err = reg.NewRegNode(node, ckPriv, skPriv)
							if err == nil {
								// DEBUG
								fmt.Printf("regNode successfully created\n")
								// END
								err = xf.MkdirsToFile(pathToConfigFile, 0700)
								if err == nil {
									err = ioutil.WriteFile(pathToConfigFile,
										[]byte(rn.String()), 0400)
									// DEBUG
								} else {
									fmt.Printf("error writing config file: %v\n",
										err.Error())
								}
								// END --------------

								// DEBUG
							} else {
								fmt.Printf("error creating regNode: %v\n",
									err.Error())
								// END
							}
						}
					}
				}
			}
		}
	}
	if err == nil {
		var r *reg.Registry
		r, err = reg.NewRegistry(nil, // nil = clusters so far
			rn, opt) // regNode, options
		if err == nil {
			logger.Printf("Registry name: %s\n", rn.GetName())
			logger.Printf("         ID:   %s\n", rn.GetNodeID().String())
		}
		if err == nil {
			var verbosity int
			if opt.Verbose {
				verbosity++
			}
			rs, err = reg.NewRegServer(r, opt.Testing, verbosity)
		}
	}
	if err != nil {
		logger.Printf("ERROR: %s\n", err.Error())
	}
	return
}
func serve(rs *reg.RegServer) (err error) {

	err = rs.Start() // non-blocking
	if err == nil {
		<-rs.DoneCh
	}

	// XXX STUB XXX

	// ORDERLY SHUTDOWN /////////////////////////////////////////////

	return
}
