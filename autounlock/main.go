package main

import (
	"os"

	"github.com/alexflint/go-arg"
	"github.com/dkaser/unraid-auto-unlock/autounlock/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

type arguments struct {
	Setup         bool   `arg:"--setup"                             help:"Setup mode"`
	Test          bool   `arg:"--test"                              help:"Run in test mode"`
	TestPath      string `arg:"--test-path"                         help:"URI to use in test mode"`
	Threshold     uint16 `arg:"--threshold"                         help:"Number of shares required to unlock drives"       default:"3"`
	Shares        uint16 `arg:"--shares"                            help:"Number of shares to split into"                   default:"5"`
	Config        string `arg:"--config"                            help:"Path to config file"                              default:"/boot/config/plugins/auto-unlock/config.txt"`
	State         string `arg:"--state"                             help:"Path to state file"                               default:"/boot/config/plugins/auto-unlock/state.json"`
	KeyFile       string `arg:"--keyfile"                           help:"Path to file to encrypt"                          default:"/root/keyfile"`
	EncryptedFile string `arg:"--encryptedfile"                     help:"Path to output encrypted file"                    default:"/boot/config/plugins/auto-unlock/unlock.enc"`
	RetryDelay    uint16 `arg:"--retry-delay,env:RETRY_DELAY"       help:"Delay between retries in seconds"                 default:"60"`
	ServerTimeout uint16 `arg:"--server-timeout,env:SERVER_TIMEOUT" help:"Timeout for server connections in seconds"        default:"30"`
	Debug         bool   `arg:"--debug"                             help:"Enable debug logging"`
	Pretty        bool   `arg:"--pretty"                            help:"Enable pretty logging output"`
	Obscure       bool   `arg:"--obscure"                           help:"Obscure a secret from stdin and output to stdout"`
}

func (arguments) Version() string {
	return version.BuildInfoString()
}

var args arguments

func main() {
	fs := afero.NewOsFs()

	arg.MustParse(&args)

	InitializeLogging()

	version.OutputToDebug()

	err := Prechecks(fs)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Prechecks failed")
		os.Exit(1)
	}

	switch {
	case args.Obscure:
		err = ObscureSecretFromStdin()
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
