package main

import "github.com/dkaser/unraid-auto-unlock/autounlock/version"

type SetupCmd struct {
	Threshold uint16 `arg:"--threshold" help:"Number of shares required to unlock drives" default:"3"`
	Shares    uint16 `arg:"--shares"    help:"Number of shares to split into"             default:"5"`
}

type ObscureCmd struct{}

type UnlockCmd struct {
	RetryDelay    uint16 `arg:"--retry-delay,env:RETRY_DELAY"       help:"Delay between retries in seconds"          default:"60"`
	ServerTimeout uint16 `arg:"--server-timeout,env:SERVER_TIMEOUT" help:"Timeout for server connections in seconds" default:"30"`
	Test          bool   `arg:"--test"                              help:"Run in test mode"`
}

type TestPathCmd struct {
	Path          string `arg:"positional,required"                 help:"URI to test"`
	ServerTimeout uint16 `arg:"--server-timeout,env:SERVER_TIMEOUT" help:"Timeout for server connections in seconds" default:"30"`
}

type ResetCmd struct {
	Force bool `arg:"--force" help:"Force reset without confirmation"`
}

type CmdArgs struct {
	Setup    *SetupCmd    `arg:"subcommand:setup"    help:"Setup auto-unlock configuration"`
	Unlock   *UnlockCmd   `arg:"subcommand:unlock"   help:"Unlock drives using auto-unlock configuration"`
	TestPath *TestPathCmd `arg:"subcommand:testpath" help:"Test access to a given path"`
	Obscure  *ObscureCmd  `arg:"subcommand:obscure"  help:"Obscure a secret read from stdin"`
	Reset    *ResetCmd    `arg:"subcommand:reset"    help:"Reset auto-unlock configuration"`

	Config        string `arg:"--config"        help:"Path to config file"       default:"/boot/config/plugins/auto-unlock/config.txt"`
	State         string `arg:"--state"         help:"Path to state file"        default:"/boot/config/plugins/auto-unlock/state.json"`
	KeyFile       string `arg:"--keyfile"       help:"Path to plaintext keyfile" default:"/root/keyfile"`
	EncryptedFile string `arg:"--encryptedfile" help:"Path to encrypted keyfile" default:"/boot/config/plugins/auto-unlock/unlock.enc"`

	Debug  bool `arg:"--debug"  help:"Enable debug logging"`
	Pretty bool `arg:"--pretty" help:"Enable pretty logging output"`
}

func (CmdArgs) Version() string {
	return version.BuildInfoString()
}
