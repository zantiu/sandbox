package pkg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/margo/dev-repo/standard/generatedCode/wfm/sbi"
)

// ConvertAppDeploymentToAppState converts AppDeployment to AppState.
func ConvertAppDeploymentToAppState(appDeployment *sbi.AppDeployment, appVersion string, appStateValue string) (sbi.AppState, error) {
	appDeploymentJSON, err := json.Marshal(appDeployment)
	if err != nil {
		return sbi.AppState{}, fmt.Errorf("error marshaling AppDeployment to JSON: %w", err)
	}

	appDeploymentYAML := base64.StdEncoding.EncodeToString(appDeploymentJSON)

	// TODO: Implement a proper hash function for appDeploymentYAML
	appDeploymentYAMLHash := "" // TODO: add a function to create hash

	appState := sbi.AppState{
		AppDeploymentYAML:     &appDeploymentYAML,
		AppDeploymentYAMLHash: appDeploymentYAMLHash,
		AppId:                 "",
		AppState:              sbi.AppStateAppState(appStateValue),
		AppVersion:            appVersion,
	}

	return appState, nil
}

// ConvertAppStateToAppDeployment converts AppState to AppDeployment.
func ConvertAppStateToAppDeployment(appState sbi.AppState) (sbi.AppDeployment, error) {
	if appState.AppDeploymentYAML == nil {
		return sbi.AppDeployment{}, fmt.Errorf("appState.AppDeploymentYAML is nil")
	}
	appDeploymentJSON, err := base64.StdEncoding.DecodeString(*appState.AppDeploymentYAML)
	if err != nil {
		return sbi.AppDeployment{}, fmt.Errorf("error decoding AppDeploymentYAML: %w", err)
	}

	var appDeployment sbi.AppDeployment
	if err := json.Unmarshal(appDeploymentJSON, &appDeployment); err != nil {
		return sbi.AppDeployment{}, fmt.Errorf("error unmarshaling AppDeployment from JSON: %w", err)
	}

	return appDeployment, nil
}

func ParseAppDeploymentFromBase64(base64Yaml string) (*sbi.AppDeployment, error) {
	decodedYaml, err := base64.StdEncoding.DecodeString(base64Yaml)
	if err != nil {
		// da.log.Errorw("Failed to decode base64 AppDeploymentYAML", "appId", state.AppId, "error", err)
		return nil, fmt.Errorf("failed to decode the app deployment yaml from its base64 format, err: %w", err)
	}

	var appDeployment sbi.AppDeployment
	if err := json.Unmarshal(decodedYaml, &appDeployment); err != nil {
		// da.log.Errorw("Failed to unmarshal JSON AppDeployment", "appId", state.AppId, "error", err)
		return nil, fmt.Errorf("failed to parse the app deployment object from the yaml, err: %w", err)
	}

	return &appDeployment, nil
}
