// Package client provides a CLI client for interacting with the Margo Device Poke API.
//
// This package offers a high-level interface for managing application packages through
// the Margo Poke Device service, including operations for onboarding, listing, retrieving,
// and deleting application packages.
package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/kr/pretty"
	pokeDeviceAPI "github.com/margo/dev-repo/standard/generatedCode/device/nbi"
)

const (
	// pokeDeviceBaseURL is the default base URL path for the Poke Device API
	pokeDeviceBaseURL = "margo/device/poke/v1"

	// Default timeout for API requests
	defaultTimeout = 30 * time.Second
)

// Type aliases for better API ergonomics
type (
	PokeDeviceReq  = pokeDeviceAPI.PokeParams
	PokeDeviceResp = any
)

// PokeDeviceCli provides a client interface for the Margo Device Poke API.
//
// This client handles HTTP communication with the Poke Device service and provides
// high-level methods for poking a device.
type PokeDeviceCli struct {
	serverAddress string
	baseURL       string
	timeout       time.Duration
	logger        *log.Logger
}

// PokeDeviceCliOption defines functional options for configuring the client
type PokeDeviceCliOption func(*PokeDeviceCli)

// WithTimeout sets a custom timeout for API requests
func WithTimeout(timeout time.Duration) PokeDeviceCliOption {
	return func(cli *PokeDeviceCli) {
		cli.timeout = timeout
	}
}

// WithLogger sets a custom logger for the client
func WithLogger(logger *log.Logger) PokeDeviceCliOption {
	return func(cli *PokeDeviceCli) {
		cli.logger = logger
	}
}

// NewPokeDeviceCli creates a new Northbound API client.
//
// Parameters:
//   - host: The hostname or IP address of the Poke Device service
//   - port: The port number of the Poke Device service
//   - basePath: Optional custom base path (uses default if nil)
//   - opts: Optional configuration functions
//
// Returns:
//   - *PokeDeviceCli: A configured client instance
//
// Example:
//
//	cli := NewPokeDeviceCli("localhost", 8080, nil,
//	    WithTimeout(60*time.Second),
//	    WithLogger(customLogger))
func NewPokeDeviceCli(host string, port uint16, basePath *string, opts ...PokeDeviceCliOption) *PokeDeviceCli {
	baseURLPath := pokeDeviceBaseURL
	if basePath != nil {
		baseURLPath = *basePath
	}

	cli := &PokeDeviceCli{
		serverAddress: fmt.Sprintf("%s:%d", host, port),
		baseURL:       fmt.Sprintf("http://%s:%d/%s", host, port, baseURLPath),
		timeout:       defaultTimeout,
		logger:        log.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(cli)
	}

	return cli
}

// createClient creates a new API client with proper error handling
func (cli *PokeDeviceCli) createClient() (*pokeDeviceAPI.Client, error) {
	client, err := pokeDeviceAPI.NewClient(cli.serverAddress, pokeDeviceAPI.WithBaseURL(cli.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return client, nil
}

// createContext creates a context with timeout
func (cli *PokeDeviceCli) createContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cli.timeout)
}

// handleErrorResponse processes error responses consistently
func (cli *PokeDeviceCli) handleErrorResponse(errBody []byte, statusCode int, operation string) error {
	// Read response body safely
	body, err := io.ReadAll(bytes.NewReader(errBody))
	if err != nil {
		cli.logger.Printf("%s request failed with error %d (could not read response body, reason: %s)", operation, statusCode, err.Error())
		return fmt.Errorf("%s failed: error (status %d) (could not read response body, reason: %s)", operation, statusCode, err.Error())
	}
	cli.logger.Printf("%s request failed with error %d: %s", operation, statusCode, string(body))
	return fmt.Errorf("%s failed: error (status %d): %s", operation, statusCode, string(body))
}

func (cli *PokeDeviceCli) PokeDevice(deviceId string, params PokeDeviceReq) (*PokeDeviceResp, error) {
	// Create client and context
	client, err := cli.createClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := cli.createContext()
	defer cancel()

	// Make API request
	resp, err := client.Poke(ctx, &params, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("onboard app package request failed: %s", err.Error())
	}
	defer resp.Body.Close()

	// Parse response
	pokeResp, err := pokeDeviceAPI.ParsePokeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse onboard app package response: %s", err.Error())
	}

	// Handle response based on status code
	switch pokeResp.StatusCode() {
	case 200, 202:
		cli.logger.Printf("Received poke device api response: %s", pretty.Sprint(pokeResp))
		return nil, nil
	default:
		return nil, cli.handleErrorResponse(pokeResp.Body, pokeResp.StatusCode(), "poke device")
	}
}
