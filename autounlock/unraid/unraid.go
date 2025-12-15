package unraid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dkaser/unraid-auto-unlock/autounlock/constants"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"gopkg.in/ini.v1"
)

// Service provides Unraid system operations.
type Service struct {
	fs afero.Fs
}

// NewService creates a new Unraid service.
func NewService(fs afero.Fs) *Service {
	return &Service{fs: fs}
}

// BlockDevices represents block device information.
type BlockDevices struct {
	BlockDevices []struct {
		Name   string `json:"name"`
		Fstype string `json:"fstype"`
	} `json:"blockdevices"`
}

// IsUnraid checks if the system is running Unraid.
func (s *Service) IsUnraid() bool {
	_, err := s.fs.Stat("/etc/unraid-version")

	return err == nil
}

// TestKeyfile tests if a keyfile can unlock LUKS devices.
func (s *Service) TestKeyfile(keyfile string) error {
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

// WaitForVarIni waits for the var.ini file to be ready.
func (s *Service) WaitForVarIni() error {
	deadline := time.Now().Add(constants.ArrayTimeout)

	for {
		_, err := s.fs.Stat("/var/local/emhttp/var.ini")
		if err == nil {
			fsState, err := s.GetFsState()
			if err == nil && fsState != "" {
				log.Debug().Str("fsState", fsState).Msg("var.ini found and readable")

				return nil
			}
		}

		if time.Now().After(deadline) {
			return errors.New("timed out waiting for var.ini to be ready")
		}

		log.Debug().
			Int("delaySeconds", int(constants.ArrayRetryDelay.Seconds())).
			Msg("var.ini not ready, retrying")
		time.Sleep(constants.ArrayRetryDelay)
	}
}

// GetFsState reads the filesystem state from var.ini.
func (s *Service) GetFsState() (string, error) {
	file, err := s.fs.Open("/var/local/emhttp/var.ini")
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

// GetCsrfToken reads the csrf token from var.ini.
func (s *Service) GetCsrfToken() (string, error) {
	file, err := s.fs.Open("/var/local/emhttp/var.ini")
	if err != nil {
		return "", fmt.Errorf("failed to open var.ini: %w", err)
	}
	defer file.Close()

	cfg, err := ini.Load(file)
	if err != nil {
		return "", fmt.Errorf("failed to read var.ini: %w", err)
	}

	csrfToken := cfg.Section("").Key("csrf_token").String()
	log.Debug().Bool("hasCsrfToken", csrfToken != "").Msg("Read csrf token from var.ini")

	if csrfToken == "" {
		return "", errors.New("csrf token is empty")
	}

	return csrfToken, nil
}

// VerifyArrayStatus checks if the array has the specified status.
func (s *Service) VerifyArrayStatus(status string) bool {
	fsState, err := s.GetFsState()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to get fsState")

		return false
	}

	return strings.EqualFold(fsState, status)
}

// StartArray starts the Unraid array.
func (s *Service) StartArray() error {
	_, err := os.Stat("/root/keyfile")
	if err != nil {
		return fmt.Errorf("keyfile not found: %w", err)
	}

	err = s.WaitForArrayStatus("Stopped", constants.ArrayStatusTimeout)
	if err != nil {
		return fmt.Errorf("array is not stopped: %w", err)
	}

	log.Info().Msg("Starting array")

	// Prepare the command parameters
	params := url.Values{}
	params.Set("startState", "STOPPED")
	params.Set("cmdStart", "Start")

	response, err := s.emhttpdCommand(params)
	if err != nil {
		return fmt.Errorf("failed to start array: %w", err)
	}

	log.Info().Str("response", response).Msg("Array start command sent successfully")

	return nil
}

// WaitForArrayStatus waits for the array to reach a specific status.
func (s *Service) WaitForArrayStatus(status string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if s.VerifyArrayStatus(status) {
			log.Debug().Str("status", status).Msg("Array has reached status")

			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for array to reach status: %s", status)
		}

		log.Debug().
			Str("desiredStatus", status).
			Int("delaySeconds", int(constants.ArrayRetryDelay.Seconds())).
			Msg("Array has not reached status yet, retrying")
		time.Sleep(constants.ArrayRetryDelay)
	}
}

// emhttpdCommand sends a command to emhttpd via Unix socket.
func (s *Service) emhttpdCommand(params url.Values) (string, error) {
	csrfToken, err := s.GetCsrfToken()
	if err != nil {
		return "", fmt.Errorf("failed to get CSRF token: %w", err)
	}

	// Add CSRF token to parameters
	params.Set("csrf_token", csrfToken)

	// Create HTTP client that communicates over Unix socket
	client := &http.Client{
		Timeout: constants.ArrayStatusTimeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}

				return d.DialContext(ctx, "unix", "/var/run/emhttpd.socket")
			},
		},
	}

	// Make the request
	req, err := http.NewRequest(
		http.MethodPost,
		"http://localhost/update",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"command failed: status %d, body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	// Check if there's an error in the response body
	responseText := strings.TrimSpace(string(body))

	return responseText, nil
}
