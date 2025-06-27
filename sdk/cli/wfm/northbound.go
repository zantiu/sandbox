// Package client provides a CLI client for interacting with the Margo Northbound API.
//
// This package offers a high-level interface for managing application packages through
// the Margo Northbound service, including operations for onboarding, listing, retrieving,
// and deleting application packages.
package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	northboundAPIClient "github.com/margo/dev-repo/sdk/api/wfm/northbound/client"
	northboundAPIModels "github.com/margo/dev-repo/sdk/api/wfm/northbound/models"
)

const (
	// northboundBaseURL is the default base URL path for the Northbound API
	northboundBaseURL = "margo/northbound/v1"

	// Default timeout for API requests
	defaultTimeout = 30 * time.Second
)

// Type aliases for better API ergonomics
type (
	AppPkgOnboardingReq  = northboundAPIModels.AppPkgOnboardingReq
	AppPkgOnboardingResp = northboundAPIModels.AppPkgOnboardingResp
	AppPkgSummary        = northboundAPIModels.AppPkgSummary
	ListAppPkgsParams    = northboundAPIModels.ListAppPkgsParams
	ListAppPkgsResp      = northboundAPIModels.ListAppPkgsResp
)

// NorthboundCli provides a client interface for the Margo Northbound API.
//
// This client handles HTTP communication with the Northbound service and provides
// high-level methods for application package management operations.
type NorthboundCli struct {
	serverAddress string
	baseURL       string
	timeout       time.Duration
	logger        *log.Logger
}

// NorthboundCliOption defines functional options for configuring the client
type NorthboundCliOption func(*NorthboundCli)

// WithTimeout sets a custom timeout for API requests
func WithTimeout(timeout time.Duration) NorthboundCliOption {
	return func(cli *NorthboundCli) {
		cli.timeout = timeout
	}
}

// WithLogger sets a custom logger for the client
func WithLogger(logger *log.Logger) NorthboundCliOption {
	return func(cli *NorthboundCli) {
		cli.logger = logger
	}
}

// NewNorthboundCli creates a new Northbound API client.
//
// Parameters:
//   - host: The hostname or IP address of the Northbound service
//   - port: The port number of the Northbound service
//   - basePath: Optional custom base path (uses default if nil)
//   - opts: Optional configuration functions
//
// Returns:
//   - *NorthboundCli: A configured client instance
//
// Example:
//
//	cli := NewNorthboundCli("localhost", 8080, nil,
//	    WithTimeout(60*time.Second),
//	    WithLogger(customLogger))
func NewNorthboundCli(host string, port uint16, basePath *string, opts ...NorthboundCliOption) *NorthboundCli {
	baseURLPath := northboundBaseURL
	if basePath != nil {
		baseURLPath = *basePath
	}

	cli := &NorthboundCli{
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
func (cli *NorthboundCli) createClient() (*northboundAPIClient.Client, error) {
	client, err := northboundAPIClient.NewClient(cli.serverAddress, northboundAPIClient.WithBaseURL(cli.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return client, nil
}

// createContext creates a context with timeout
func (cli *NorthboundCli) createContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cli.timeout)
}

// handleErrorResponse processes error responses consistently
func (cli *NorthboundCli) handleErrorResponse(errBody []byte, statusCode int, operation string) error {
	// Read response body safely
	body, err := io.ReadAll(bytes.NewReader(errBody))
	if err != nil {
		cli.logger.Printf("%s request failed with error %d (could not read response body, reason: %s)", operation, statusCode, err.Error())
		return fmt.Errorf("%s failed: error (status %d) (could not read response body, reason: %s)", operation, statusCode, err.Error())
	}
	cli.logger.Printf("%s request failed with error %d: %s", operation, statusCode, string(body))
	return fmt.Errorf("%s failed: error (status %d): %s", operation, statusCode, string(body))
}

// OnboardAppPkg onboards a new application package.
//
// This method validates the request parameters and submits an onboarding request
// to the Northbound service. The service will process the package from the specified
// source and make it available for deployment.
//
// Parameters:
//   - params: The onboarding request parameters including name, source type, and source details
//
// Returns:
//   - *AppPkgOnboardingResp: The onboarding response with package details
//   - error: An error if validation fails or the request cannot be processed
//
// Example:
//
//	req := AppPkgOnboardingReq{
//	    Name: "my-app",
//	    SourceType: "git",
//	    Source: map[string]interface{}{"url": "https://github.com/user/app.git"},
//	}
//	resp, err := cli.OnboardAppPkg(req)
func (cli *NorthboundCli) OnboardAppPkg(params AppPkgOnboardingReq) (*AppPkgOnboardingResp, error) {
	// Validate required parameters
	if params.Name == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}
	if params.SourceType == "" {
		return nil, fmt.Errorf("source type cannot be empty")
	}

	// Create client and context
	client, err := cli.createClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := cli.createContext()
	defer cancel()

	// Make API request
	resp, err := client.OnboardAppPkg(ctx, northboundAPIModels.OnboardAppPkgJSONRequestBody{
		Name:       params.Name,
		SourceType: params.SourceType,
		Source:     params.Source,
	})
	if err != nil {
		return nil, fmt.Errorf("onboard app package request failed: %s", err.Error())
	}
	defer resp.Body.Close()

	// Parse response
	pkgResp, err := northboundAPIClient.ParseOnboardAppPkgResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse onboard app package response: %s", err.Error())
	}

	// Handle response based on status code
	switch pkgResp.StatusCode() {
	case 202:
		cli.logger.Printf("Application onboard request accepted for package: %s", params.Name)
		if pkgResp.JSON202 != nil && pkgResp.JSON202.Data != nil {
			return pkgResp.JSON202.Data, nil
		}
		return nil, nil
	default:
		return nil, cli.handleErrorResponse(pkgResp.Body, pkgResp.StatusCode(), "onboard app package")
	}
}

// GetAppPkg retrieves details for a specific application package.
//
// Parameters:
//   - pkgId: The unique identifier of the package to retrieve
//
// Returns:
//   - *AppPkgSummary: The package summary with details
//   - error: An error if the package is not found or cannot be retrieved
func (cli *NorthboundCli) GetAppPkg(pkgId string) (*AppPkgSummary, error) {
	if pkgId == "" {
		return nil, fmt.Errorf("package ID cannot be empty")
	}

	client, err := cli.createClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := cli.createContext()
	defer cancel()

	resp, err := client.GetAppPkg(ctx, pkgId)
	if err != nil {
		return nil, fmt.Errorf("get app package request failed: %w", err)
	}
	defer resp.Body.Close()

	pkgResp, err := northboundAPIClient.ParseGetAppPkgResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse get app package response: %w", err)
	}

	switch pkgResp.StatusCode() {
	case 200:
		cli.logger.Printf("Successfully retrieved package: %s", pkgId)
		return pkgResp.JSON200.Data, nil
	default:
		return nil, cli.handleErrorResponse(pkgResp.Body, pkgResp.StatusCode(), "get app package")
	}
}

// ListAppPkgs retrieves a list of application packages.
//
// Parameters:
//   - params: Optional filtering and pagination parameters
//
// Returns:
//   - *ListAppPkgsResp: The list response containing packages and metadata
//   - error: An error if the request cannot be processed
func (cli *NorthboundCli) ListAppPkgs(params ListAppPkgsParams) (*ListAppPkgsResp, error) {
	client, err := cli.createClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := cli.createContext()
	defer cancel()

	resp, err := client.ListAppPkgs(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("list app packages request failed: %w", err)
	}
	defer resp.Body.Close()

	pkgResp, err := northboundAPIClient.ParseListAppPkgsResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list app packages response: %w", err)
	}

	switch pkgResp.StatusCode() {
	case 200:
		packageCount := 0
		if pkgResp.JSON200.Data != nil && pkgResp.JSON200.Data.AppPkgs != nil {
			packageCount = len(pkgResp.JSON200.Data.AppPkgs)
		}
		cli.logger.Printf("Successfully listed %d packages", packageCount)
		return pkgResp.JSON200.Data, nil
	default:
		return nil, cli.handleErrorResponse(pkgResp.Body, pkgResp.StatusCode(), "list app packages")
	}
}

// DeleteAppPkg deletes a specific application package.
//
// Parameters:
//   - pkgId: The unique identifier of the package to delete
//
// Returns:
//   - error: An error if the package cannot be deleted
func (cli *NorthboundCli) DeleteAppPkg(pkgId string) error {
	if pkgId == "" {
		return fmt.Errorf("package ID cannot be empty")
	}

	client, err := cli.createClient()
	if err != nil {
		return err
	}

	ctx, cancel := cli.createContext()
	defer cancel()

	resp, err := client.DeleteAppPkg(ctx, pkgId, &northboundAPIModels.DeleteAppPkgParams{})
	if err != nil {
		return fmt.Errorf("delete app package request failed: %w", err)
	}
	defer resp.Body.Close()

	pkgResp, err := northboundAPIClient.ParseDeleteAppPkgResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to parse delete app package response: %w", err)
	}

	switch pkgResp.StatusCode() {
	case 200, 202:
		cli.logger.Printf("Successfully deleted package: %s", pkgId)
		return nil
	default:
		return cli.handleErrorResponse(pkgResp.Body, pkgResp.StatusCode(), "delete app package")
	}
}
