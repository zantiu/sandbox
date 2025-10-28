package main

import (
	"context"
	"time"

	"github.com/kr/pretty"
	"github.com/margo/dev-repo/poc/device/agent/database"
	wfm "github.com/margo/dev-repo/poc/wfm/cli"
	"github.com/margo/dev-repo/standard/generatedCode/wfm/sbi"
	"go.uber.org/zap"
)

type StatusReporterIfc interface {
	Start()
	Stop()
}

type StatusReporter struct {
	database  database.DatabaseIfc
	apiClient wfm.SBIAPIClientInterface
	deviceID  string
	log       *zap.SugaredLogger
	stopChan  chan struct{}
}

func NewStatusReporter(db database.DatabaseIfc, client wfm.SBIAPIClientInterface, deviceID string, log *zap.SugaredLogger) *StatusReporter {
	return &StatusReporter{
		database:  db,
		apiClient: client,
		deviceID:  deviceID,
		log:       log,
		stopChan:  make(chan struct{}),
	}
}

func (sr *StatusReporter) Start() {
	// Subscribe to database changes for status updates
	sr.database.Subscribe(sr.onDeploymentChange)
}

func (sr *StatusReporter) Stop() {
	close(sr.stopChan)
}

func (sr *StatusReporter) onDeploymentChange(appID string, record *database.DeploymentRecord, changeType database.DeploymentRecordChangeType) {
	sr.log.Info("onDeploymentChange", "msg", "event received", "record", pretty.Sprint(record))
	// Report status when phase changes
	if changeType == database.DeploymentChangeTypeDesiredStateAdded ||
		changeType == database.DeploymentChangeTypeComponentPhaseChanged {
		go sr.reportStatus(appID, record)
	}
}

func (sr *StatusReporter) reportStatus(appID string, record *database.DeploymentRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert component status
	components := make([]sbi.ComponentStatus, 0, len(record.ComponentStatus))
	for _, status := range record.ComponentStatus {
		components = append(components, status)
	}

	deploymentStatus := sbi.OverallStatus{
		State: sbi.OverallStatusState(record.Phase),
	}

	err := sr.apiClient.ReportDeploymentStatus(ctx, sr.deviceID, appID, deploymentStatus, components)
	if err != nil {
		sr.log.Errorw("Failed to report status", "appId", appID, "error", err)
		return
	}

	sr.log.Debugw("Status reported", "appId", appID, "phase", record.Phase)
}
