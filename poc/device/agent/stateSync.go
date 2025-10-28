// sync/state_syncer.go
package main

import (
	"context"
	"crypto"
	"fmt"
	"time"

	"github.com/margo/dev-repo/poc/device/agent/database"
	wfm "github.com/margo/dev-repo/poc/wfm/cli"
	"github.com/margo/dev-repo/shared-lib/http/auth"
	"github.com/margo/dev-repo/standard/generatedCode/wfm/sbi"
	"github.com/margo/dev-repo/standard/pkg"
	"go.uber.org/zap"
)

type StateSyncerIfc interface {
	Start()
	Stop()
}

type StateSyncer struct {
	database                  *database.Database
	apiClient                 wfm.SBIAPIClientInterface
	requestSigner             crypto.Signer
	deviceID                  string
	log                       *zap.SugaredLogger
	stopChan                  chan struct{}
	stateSyncingIntervalInSec uint16
}

func NewStateSyncer(
	db *database.Database,
	client wfm.SBIAPIClientInterface,
	deviceID string,
	stateSeekingIntervalInSec uint16,
	log *zap.SugaredLogger) *StateSyncer {
	return &StateSyncer{
		database:                  db,
		apiClient:                 client,
		deviceID:                  deviceID,
		log:                       log,
		stopChan:                  make(chan struct{}),
		stateSyncingIntervalInSec: stateSeekingIntervalInSec,
	}
}

func (ss *StateSyncer) Start() {
	go ss.syncLoop()
}

func (ss *StateSyncer) Stop() {
	close(ss.stopChan)
}

func (ss *StateSyncer) syncLoop() {
	ticker := time.NewTicker(time.Duration(ss.stateSyncingIntervalInSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ss.performSync()
		case <-ss.stopChan:
			return
		}
	}
}

func (ss *StateSyncer) performSync() {
	ss.log.Debugf("performing sync....")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get current states from database
	deployments := ss.database.ListDeployments()
	currentStates := make([]sbi.AppState, 0, len(deployments))

	for _, deployment := range deployments {
		if deployment.CurrentState != nil {
			currentStates = append(currentStates, *deployment.CurrentState)
		}
	}

	// Send to orchestrator and get desired states
	device, err := ss.database.GetDeviceSettings()
	if err != nil {
		ss.log.Errorw("Sync failed", "err", err.Error(), "msg", "failed to fetch the device settings")
		return
	}
	var desiredStates sbi.DesiredAppStates
	if device.AuthEnabled {
		desiredStates, err = ss.apiClient.SyncState(
			ctx,
			device.DeviceClientId,
			currentStates,
			auth.WithOAuth(ctx, device.OAuthClientId, device.OAuthClientSecret, device.OAuthTokenEndpointUrl),
		)
	} else {
		desiredStates, err = ss.apiClient.SyncState(
			ctx,
			device.DeviceClientId,
			currentStates,
		)
	}
	if err != nil {
		ss.log.Errorw("Sync failed", "err", err.Error())
		return
	}

	// Update desired states in database
	ss.log.Debugf("setting desired states....")
	for _, desiredState := range desiredStates {
		appDeployment, err := pkg.ConvertAppStateToAppDeployment(desiredState)
		if err != nil {
			ss.database.SetPhase(desiredState.AppId, "FAILED", fmt.Sprintf("Conversion failed: %v", err))
			return
		}
		// if !ss.database.CanDeployAppProfile(string(appDeployment.Spec.DeploymentProfile.Type)) {
		// 	ss.log.Warnw("Received unsupported file type for this agent/runtime, will skip it", "profileType", appDeployment.Spec.DeploymentProfile.Type)
		// 	continue
		// }
		deploymentId := appDeployment.Metadata.Id
		ss.database.SetDesiredState(*deploymentId, desiredState)
	}

	ss.log.Debugw("Sync completed", "desiredStates", len(desiredStates))
}
