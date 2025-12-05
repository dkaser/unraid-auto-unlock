package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/dkaser/unraid-auto-unlock/autounlock/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

var args struct {
	Setup         bool   `arg:"--setup"          help:"Setup mode"`
	Test          bool   `arg:"--test"           help:"Run in test mode"`
	TestPath      string `arg:"--test-path"      help:"URI to use in test mode"`
	Threshold     uint16 `arg:"--threshold"      default:"3"                                           help:"Threshold for setup mode"`
	Shares        uint16 `arg:"--shares"         default:"5"                                           help:"Number of shares to split into"`
	Config        string `arg:"--config"         default:"/boot/config/plugins/auto-unlock/config.txt" help:"Path to config file"`
	State         string `arg:"--state"          default:"/boot/config/plugins/auto-unlock/state.json" help:"Path to state file"`
	KeyFile       string `arg:"--keyfile"        default:"/root/keyfile"                               help:"Path to file to encrypt"`
	EncryptedFile string `arg:"--encryptedfile"  default:"/boot/config/plugins/auto-unlock/unlock.enc" help:"Path to output encrypted file"`
	RetryDelay    uint16 `arg:"--retry-delay"    default:"60"                                          help:"Delay between retries in seconds"`
	ServerTimeout uint16 `arg:"--server-timeout" default:"30"                                          help:"Timeout for server connections in seconds"`
	Debug         bool   `arg:"--debug"          help:"Enable debug logging"`
	Pretty        bool   `arg:"--pretty"         help:"Enable pretty logging output"`
	Version       bool   `arg:"--version"        help:"Show version information"`
}

func main() {
	fs := afero.NewOsFs()

	err := arg.Parse(&args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	InitializeLogging()

	if args.Version {
		fmt.Print(version.BuildInfoString())

		return
	}

	version.OutputToDebug()

	err = Prechecks(fs)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Prechecks failed")
		os.Exit(1)
	}

	switch {
	case args.Setup:
		err = Setup(fs)
	case args.TestPath != "":
		err = TestPath(fs)
	default:
		err = Unlock(fs)
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
