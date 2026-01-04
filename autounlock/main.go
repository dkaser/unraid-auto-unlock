package main

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

func main() {
	fs := afero.NewOsFs()

	args := parseArgs()

	if args.License != nil {
		printLicense()

		return
	}

	autoUnlock, err := NewAutoUnlock(fs, args)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed to initialize AutoUnlock")
	}

	lockFile, err := lockApp()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Another instance of the application is already running")
	}
	defer lockFile.Close()

	switch {
	case args.Reset != nil:
		err = autoUnlock.ResetConfiguration()
	case args.Obscure != nil:
		err = autoUnlock.ObscureSecretFromStdin()
	case args.Setup != nil:
		err = autoUnlock.Setup()
	case args.TestPath != nil:
		err = autoUnlock.TestPath()
	case args.Unlock != nil:
		err = autoUnlock.Unlock()
	}

	if err != nil {
		lockFile.Close()
		log.Fatal().Stack().Err(err).Msg("Failed to execute command") //nolint:gocritic
	}
}

func printLicense() {
	licenseText := `
	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	`

	fmt.Println(licenseText)
}
