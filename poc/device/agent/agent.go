package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/kr/pretty"
	"github.com/margo/dev-repo/standard/generatedCode/wfm/sbi"
	"github.com/margo/dev-repo/standard/pkg"
	"go.uber.org/zap"
)

const (
	stateSeekURLPath     = "/wfm/state" // Corrected naming
	appStateSyncInterval = 5 * time.Second
)

// DeviceAgent represents the main device agent.
type DeviceAgent struct {
	config *Config // Renamed AgentConfig to Config
	device *Device // Device information
	// server           *http.Server            // HTTP server for the agent
	ctx              context.Context         // Context for managing agent lifecycle
	cancelFunc       context.CancelFunc      // Function to cancel the context
	currentWorkloads map[string]sbi.AppState // workload id -> AppState // Corrected type and added comment
	log              *zap.SugaredLogger      // Structured logger
	validator        *validator.Validate     // Validator instance
}

// Config holds configuration for the device agent.
type Config struct { // Renamed AgentConfig to Config
	DeviceID  string `json:"deviceId" validate:"required"`      // Device identifier
	WfmSbiUrl string `json:"wfmServer" validate:"required,url"` // WFM server "https://host:port/margo/sbi" ensure to include the base path
}

// Device represents the device instance.
type Device struct {
	ID              string       // Device identifier
	Status          DeviceStatus // Device operational status
	AppCapabilities []string     // Application capabilities
	LastSeen        time.Time    // Last seen timestamp
}

// DeviceStatus represents device operational status.
type DeviceStatus string

const (
	StatusOffline DeviceStatus = "offline" // Device is offline
	StatusOnline  DeviceStatus = "online"  // Device is online
	StatusError   DeviceStatus = "error"   // Device is in an error state
)

var (
	ErrFailedToCreateClient = fmt.Errorf("failed to create API client") // Predefined error
	ErrSyncFailed           = fmt.Errorf("failed to sync app states")
)

// NewDeviceAgent creates a new device agent instance.
func NewDeviceAgent(config *Config, logger *zap.SugaredLogger) (*DeviceAgent, error) { // Renamed AgentConfig to Config
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	validate := validator.New()
	err := validate.Struct(config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DeviceAgent{
		config:           config,
		ctx:              ctx,
		cancelFunc:       cancel,
		currentWorkloads: make(map[string]sbi.AppState), // Initialize the map
		log:              logger,
		validator:        validate,
	}, nil
}

// Start initializes and starts the device agent.
func (da *DeviceAgent) Start() error {
	da.log.Info("Starting device agent...")

	device, err := da.newDevice()
	if err != nil {
		return fmt.Errorf("failed to initialize device: %w", err)
	}
	da.device = device

	go da.startStateMonitoring()

	da.log.Infow("Device agent started successfully", "deviceID", da.device.ID)
	return nil
}

// Stop gracefully shuts down the device agent.
func (da *DeviceAgent) Stop() error {
	da.log.Info("Stopping device agent...")

	da.cancelFunc()

	// if da.server != nil {
	// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// 	defer cancel()
	// 	if err := da.server.Shutdown(ctx); err != nil {
	// 		da.log.Errorw("Failed to shutdown server gracefully", "error", err)
	// 		return err
	// 	}
	// 	da.log.Info("Server stopped gracefully")
	// }

	da.log.Info("Device agent stopped")
	return nil
}

// newDevice creates a new device instance with credentials.
func (da *DeviceAgent) newDevice() (*Device, error) {
	if da.config == nil {
		return nil, fmt.Errorf("device agent config is nil")
	}
	device := &Device{
		ID:              da.config.DeviceID,
		Status:          StatusOnline,
		AppCapabilities: []string{"helm", "docker-compose"},
		LastSeen:        time.Now(),
	}
	da.log.Infow("New device created", "deviceID", device.ID)
	return device, nil
}

// startStateMonitoring starts device state monitoring.
func (da *DeviceAgent) startStateMonitoring() {
	da.log.Info("Starting state monitoring...")
	ticker := time.NewTicker(appStateSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-da.ctx.Done():
			da.log.Info("State monitoring stopped due to context cancellation")
			return
		case <-ticker.C:
			if err := da.syncAppStates(); err != nil {
				da.log.Errorw("Failed to sync app states", "error", err)
			}
		}
	}
}

// syncAppStates checks the state of apps from the device, hits the state seeking API of the server, and then syncs the state.
func (da *DeviceAgent) syncAppStates() error {
	da.log.Debug("Syncing app states...")

	currentAppStates := make(sbi.StateJSONRequestBody, 0, len(da.currentWorkloads)) // Pre-allocate slice
	for _, appState := range da.currentWorkloads {
		currentAppStates = append(currentAppStates, appState)
	}

	client, err := sbi.NewClient(da.config.WfmSbiUrl)
	if err != nil {
		da.log.Errorw("Failed to create API client", "error", err)
		return fmt.Errorf("%w: %v", ErrFailedToCreateClient, err) // Use predefined error
	}

	resp, err := client.State(da.ctx, currentAppStates)
	if err != nil {
		da.log.Errorw("Failed to send state sync request", "error", err)
		return fmt.Errorf("failed to prepare state sync request: %w", err)
	}
	defer resp.Body.Close()

	desiredStateResp, err := sbi.ParseStateResponse(resp)
	if err != nil {
		da.log.Errorw("Failed to parse app state response", "error", err)
		return fmt.Errorf("failed to parse app state response: %w", err)
	}

	switch desiredStateResp.StatusCode() {
	case http.StatusOK, http.StatusAccepted:
		da.log.Debugw("Received desired state API response", "response", pretty.Sprint(desiredStateResp.JSON200))
		if desiredStateResp.JSON200 != nil {
			if err := da.mergeAppStates(*desiredStateResp.JSON200); err != nil {
				da.log.Errorw("Failed to merge app states", "error", err)
				return err
			}
		}
		da.log.Debug("App states synced successfully")
		return nil
	default:
		return da.handleErrorResponse(desiredStateResp.Body, desiredStateResp.StatusCode(), "sync app state")
	}
}

// handleErrorResponse processes error responses consistently.
func (da *DeviceAgent) handleErrorResponse(errBody []byte, statusCode int, operation string) error {
	body, err := io.ReadAll(bytes.NewReader(errBody))
	if err != nil {
		da.log.Errorw("Failed to read response body", "operation", operation, "statusCode", statusCode, "error", err)
		return fmt.Errorf("%s request failed with status %d (could not read response body): %w", operation, statusCode, err)
	}

	da.log.Errorw("Request failed", "operation", operation, "statusCode", statusCode, "body", string(body))
	return fmt.Errorf("%s request failed with status %d: %s", operation, statusCode, string(body))
}

// mergeAppStates merges the desired app states with the current app states.
func (da *DeviceAgent) mergeAppStates(states sbi.DesiredAppStates) error {
	da.log.Debugw("Merging desired app states", "desiredStates", pretty.Sprint(states))

	for _, state := range states {
		if state.AppDeploymentYAML == nil {
			da.log.Warnw("Received state with nil AppDeploymentYAML, skipping", "appId", state.AppId)
			continue
		}

		appDeployment, err := pkg.ParseAppDeploymentFromBase64(*state.AppDeploymentYAML)
		if err != nil {
			return fmt.Errorf("failed to parse the deployment yaml from the base64 encoded string, err: %s", err.Error())
		}

		if appDeployment.Metadata.Id == nil {
			da.log.Warnw("Received AppDeployment with nil ID in the metadata field, skipping", "appId", state.AppId)
			continue
		}
		da.log.Debugw("Successfully processed AppDeployment", "name", appDeployment.Metadata.Name, "id", *appDeployment.Metadata.Id)

		switch state.AppState {
		case sbi.REMOVING:
			da.log.Infow("Removing app", "appId", *appDeployment.Metadata.Id)
			delete(da.currentWorkloads, *appDeployment.Metadata.Id)

		case sbi.RUNNING, sbi.UPDATING:
			da.log.Infow("Adding/Updating app", "appId", *appDeployment.Metadata.Id, "state", state.AppState)
			newAppState, err := pkg.ConvertAppDeploymentToAppState(appDeployment, *appDeployment.Metadata.Id, "v1", "RUNNING")
			if err != nil {
				da.log.Errorw("Failed to convert AppDeployment to AppState", "appId", *appDeployment.Metadata.Id, "error", err)
				return err
			}
			da.currentWorkloads[*appDeployment.Metadata.Id] = newAppState

		default:
			da.log.Warnw("Unknown app state, skipping", "appId", *appDeployment.Metadata.Id, "state", state.AppState)
		}
	}

	da.log.Debug("App states merged successfully")
	return nil
}
