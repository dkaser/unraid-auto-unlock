package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
)

type BlockDevices struct {
	BlockDevices []struct {
		Name   string `json:"name"`
		Fstype string `json:"fstype"`
	} `json:"blockdevices"`
}

func TestKeyfile() error {
	log.Info().Str("keyfile", args.KeyFile).Msg("Verifying that key can unlock disks")

	_, err := os.Stat(args.KeyFile)
	if err != nil {
		return fmt.Errorf("keyfile not found: %w", err)
	}

	log.Debug().Str("keyfile", args.KeyFile).Msg("Keyfile exists")

	out, err := exec.Command("/bin/lsblk", "-Jpo", "NAME,FSTYPE", "-Q", "FSTYPE=='crypto_LUKS'").
		Output()
	if err != nil {
		return fmt.Errorf("failed to run lsblk: %w", err)
	}

	var devices BlockDevices

	err = json.Unmarshal(out, &devices)
	if err != nil {
		return fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	for _, device := range devices.BlockDevices {
		log.Debug().
			Str("device", device.Name).
			Str("fstype", device.Fstype).
			Msg("Found block device")

		log.Info().Str("device", device.Name).Msg("LUKS encrypted device found")

		cmd := exec.Command( // #nosec G204
			"/sbin/cryptsetup",
			"luksOpen",
			"--test-passphrase",
			"--key-file",
			args.KeyFile,
			device.Name,
		)

		err := cmd.Run()
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Str("device", device.Name).
				Msg("Failed to unlock LUKS device")

			continue
		}

		log.Info().Str("device", device.Name).Msg("LUKS device unlocked successfully")

		return nil
	}

	return errors.New("keyfile could not decrypt any LUKS devices")
}

func VerifyArrayStopped() bool {
	cfg, err := ini.Load("/var/local/emhttp/var.ini")
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to read var.ini")

		return false
	}

	fsState := cfg.Section("").Key("fsState").String()
	log.Debug().Str("fsState", fsState).Msg("Read fsState from var.ini")

	return fsState == "Stopped"
}

func StartArray() error {
	_, err := os.Stat("/root/keyfile")
	if err != nil {
		return fmt.Errorf("keyfile not found: %w", err)
	}

	if !VerifyArrayStopped() {
		return errors.New("array is not stopped")
	}

	log.Info().Msg("Starting array")

	osCmd := "/usr/local/sbin/emcmd"
	args := []string{"startState=STOPPED&cmdStart=Start"}

	cmd := exec.Command(osCmd, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start array: %w, output: %s", err, string(output))
	}

	log.Info().Msg("Array started successfully")

	return nil
}
