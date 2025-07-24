// Package client provides a CLI client for interacting with the Margo DesiredState API.
//
// This package offers a high-level interface for managing application packages through
// the Margo DesiredState service, including operations for onboarding, listing, retrieving,
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
	wfmSBI "github.com/margo/dev-repo/standard/generatedCode/wfm/sbi"
)

const (
	// desiredStateBaseURL is the default base URL path for the DesiredState API
	desiredStateBaseURL = "margo/desiredState/v1"

	// Default timeout for API requests
	defaultTimeout = 30 * time.Second
)

// Type aliases for better API ergonomics
type (
	SyncAppStateReq  = wfmSBI.StateJSONRequestBody
	SyncAppStateResp = wfmSBI.DesiredAppStates
)

// DesiredStateCli provides a client interface for the Margo DesiredState API.
//
// This client handles HTTP communication with the DesiredState service and provides
// high-level methods for desired state seeking operation.
type DesiredStateCli struct {
	serverAddress string
	baseURL       string
	timeout       time.Duration
	logger        *log.Logger
}

// DesiredStateCliOption defines functional options for configuring the client
type DesiredStateCliOption func(*DesiredStateCli)

// WithTimeout sets a custom timeout for API requests
func WithTimeout(timeout time.Duration) DesiredStateCliOption {
	return func(cli *DesiredStateCli) {
		cli.timeout = timeout
	}
}

// WithLogger sets a custom logger for the client
func WithLogger(logger *log.Logger) DesiredStateCliOption {
	return func(cli *DesiredStateCli) {
		cli.logger = logger
	}
}

// NewDesiredStateCli creates a new DesiredState API client.
//
// Parameters:
//   - host: The hostname or IP address of the DesiredState service
//   - port: The port number of the DesiredState service
//   - basePath: Optional custom base path (uses default if nil)
//   - opts: Optional configuration functions
//
// Returns:
//   - *DesiredStateCli: A configured client instance
//
// Example:
//
//	cli := NewDesiredStateCli("localhost", 8080, nil,
//	    WithTimeout(60*time.Second),
//	    WithLogger(customLogger))
func NewDesiredStateCli(host string, port uint16, basePath *string, opts ...DesiredStateCliOption) *DesiredStateCli {
	baseURLPath := desiredStateBaseURL
	if basePath != nil {
		baseURLPath = *basePath
	}

	cli := &DesiredStateCli{
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
func (cli *DesiredStateCli) createClient() (*wfmSBI.Client, error) {
	client, err := wfmSBI.NewClient(cli.serverAddress, wfmSBI.WithBaseURL(cli.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return client, nil
}

// createContext creates a context with timeout
func (cli *DesiredStateCli) createContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cli.timeout)
}

// handleErrorResponse processes error responses consistently
func (cli *DesiredStateCli) handleErrorResponse(errBody []byte, statusCode int, operation string) error {
	// Read response body safely
	body, err := io.ReadAll(bytes.NewReader(errBody))
	if err != nil {
		cli.logger.Printf("%s request failed with error %d (could not read response body, reason: %s)", operation, statusCode, err.Error())
		return fmt.Errorf("%s failed: error (status %d) (could not read response body, reason: %s)", operation, statusCode, err.Error())
	}
	cli.logger.Printf("%s request failed with error %d: %s", operation, statusCode, string(body))
	return fmt.Errorf("%s failed: error (status %d): %s", operation, statusCode, string(body))
}

func (cli *DesiredStateCli) SyncAppState(deviceId string, params SyncAppStateReq) (*SyncAppStateResp, error) {
	// Create client and context
	client, err := cli.createClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := cli.createContext()
	defer cancel()

	// Make API request
	resp, err := client.State(ctx,
		&wfmSBI.StateParams{
			DeviceId: &deviceId,
		}, params, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("onboard app package request failed: %s", err.Error())
	}
	defer resp.Body.Close()

	// Parse response
	desiredStateResp, err := wfmSBI.ParseStateResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse onboard app package response: %s", err.Error())
	}

	// Handle response based on status code
	switch desiredStateResp.StatusCode() {
	case 200, 202:
		cli.logger.Printf("Received desired state api response: %s", pretty.Sprint(desiredStateResp))
		if desiredStateResp.JSON200 != nil {
			return desiredStateResp.JSON200, nil
		}
		return nil, nil
	default:
		return nil, cli.handleErrorResponse(desiredStateResp.Body, desiredStateResp.StatusCode(), "sync app state")
	}
}
