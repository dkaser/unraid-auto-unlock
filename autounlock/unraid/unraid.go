package unraid

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"gopkg.in/ini.v1"
)

const (
	arrayRetryDelay = 15 * time.Second
	arrayTimeout    = 15 * time.Minute
)

type BlockDevices struct {
	BlockDevices []struct {
		Name   string `json:"name"`
		Fstype string `json:"fstype"`
	} `json:"blockdevices"`
}

func IsUnraid(fs afero.Fs) bool {
	_, err := fs.Stat("/etc/unraid-version")

	return err == nil
}

func TestKeyfile(keyfile string) error {
	log.Info().Str("keyfile", keyfile).Msg("Verifying that key can unlock disks")

	_, err := os.Stat(keyfile)
	if err != nil {
		return fmt.Errorf("keyfile not found: %w", err)
	}

	log.Debug().Str("keyfile", keyfile).Msg("Keyfile exists")

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
			keyfile,
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

func WaitForVarIni(fs afero.Fs) error {
	deadline := time.Now().Add(arrayTimeout)

	for {
		_, err := fs.Stat("/var/local/emhttp/var.ini")
		if err == nil {
			fsState, err := GetFsState(fs)
			if err == nil && fsState != "" {
				log.Debug().Str("fsState", fsState).Msg("var.ini found and readable")

				return nil
			}
		}

		if time.Now().After(deadline) {
			return errors.New("timed out waiting for var.ini to be ready")
		}

		log.Debug().
			Int("delaySeconds", int(arrayRetryDelay.Seconds())).
			Msg("var.ini not ready, retrying")
		time.Sleep(arrayRetryDelay)
	}
}

func GetFsState(fs afero.Fs) (string, error) {
	file, err := fs.Open("/var/local/emhttp/var.ini")
	if err != nil {
		return "", fmt.Errorf("failed to open var.ini: %w", err)
	}
	defer file.Close()

	cfg, err := ini.Load(file)
	if err != nil {
		return "", fmt.Errorf("failed to read var.ini: %w", err)
	}

	fsState := cfg.Section("").Key("fsState").String()
	log.Debug().Str("fsState", fsState).Msg("Read fsState from var.ini")

	return fsState, nil
}

func VerifyArrayStatus(fs afero.Fs, status string) bool {
	fsState, err := GetFsState(fs)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to get fsState")

		return false
	}

	return strings.EqualFold(fsState, status)
}

func StartArray(fs afero.Fs) error {
	_, err := os.Stat("/root/keyfile")
	if err != nil {
		return fmt.Errorf("keyfile not found: %w", err)
	}

	if !VerifyArrayStatus(fs, "Stopped") {
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

func WaitForArrayStarted(fs afero.Fs) error {
	deadline := time.Now().Add(arrayTimeout)

	for {
		if VerifyArrayStatus(fs, "Started") {
			log.Debug().Msg("Array has started")

			return nil
		}

		if time.Now().After(deadline) {
			return errors.New("timed out waiting for array to start")
		}

		log.Debug().
			Int("delaySeconds", int(arrayRetryDelay.Seconds())).
			Msg("Array not started yet, retrying")
		time.Sleep(arrayRetryDelay)
	}
}
