package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

// See loadConfig for details on the configuration load process.
type config struct {
	RPCPort int `long:"rpcport" short:"p" description:"Linsten port of RPC server"`
}

// newConfigParser returns a new command line flags parser.
func newConfigParser(cfg *config, options flags.Options) *flags.Parser {
	parser := flags.NewParser(cfg, options)
	return parser
}

func loadConfig() (*config, error) {
	cfg := config{
		RPCPort: rpcServerPort,
	}

	//preCfg := cfg
	preParser := newConfigParser(&cfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			return nil, err
		}
	}

	return &cfg, nil
}
