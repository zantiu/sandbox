package wfm

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
	"github.com/google/uuid"
	"github.com/margo/dev-repo/shared-lib/pointers"
	"github.com/margo/dev-repo/standard/generatedCode/wfm/sbi"
	"crypto/sha256"
	"io"

)

const (
	// southboundBaseURL is the default base URL path for the Northbound API
	southboundBaseURL = "margo/sbi/v1"

	// Default timeout for API requests
	sbiDefaultTimeout = 30 * time.Second
)

type HTTPApiClientRequestEditorOptions = sbi.RequestEditorFn
type HTTPApiClientOptions = sbi.ClientOption

// SbiHttpClient implementation
type SbiHttpClient struct {
	// You can add any dependencies needed for creating the client here, like auth info
	// clientId, clientSecret, tokenUrl string
	url     string
	client  sbi.ClientInterface
	options []HTTPApiClientOptions
}

func NewSbiHTTPClient(url string, options ...HTTPApiClientOptions) (*SbiHttpClient, error) {
	client, err := sbi.NewClient(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	for _, opt := range options {
		opt(client)
	}

	apiClient := &SbiHttpClient{
		url:     url,
		client:  client,
		options: options,
	}
	return apiClient, nil
}

func (self *SbiHttpClient) OnboardDeviceClient(ctx context.Context, deviceCertificate []byte, overrideOptions ...HTTPApiClientRequestEditorOptions) (clientId string, endpoints []string, err error) {
	cert := base64.StdEncoding.EncodeToString([]byte(deviceCertificate))

	onboardingReq := sbi.PostApiV1OnboardingJSONRequestBody{
		PublicCertificate: &cert,
	}

	resp, err := self.client.PostApiV1Onboarding(ctx, onboardingReq, overrideOptions...)
	if err != nil {
		return "", nil, fmt.Errorf("onboarding failed: %w", err)
	}
	defer resp.Body.Close()

	     

	if resp.StatusCode != 201 {
		return "", nil, fmt.Errorf("onboarding failed with status: %d", resp.StatusCode)
	}

	onboardingResp, err := sbi.ParsePostApiV1OnboardingResponse(resp)
	if err != nil {
		return "", nil, fmt.Errorf("onboarding device response parsing failed: %w", err)
	}

	if onboardingResp.JSON201 == nil {
		return "", nil, fmt.Errorf("unexpected response format: JSON201 is nil")
	}

	if onboardingResp.JSON201.ClientId == nil {
		return "", nil, fmt.Errorf("clientId is nil in the onboarding response")
	}

	if *onboardingResp.JSON201.ClientId == "" {
		return "", nil, fmt.Errorf("the clientid is empty in the onboarding response, this should never happen!")
	}

	var endpointsList []string
	return *onboardingResp.JSON201.ClientId, endpointsList, nil
}

func (self *SbiHttpClient) ReportCapabilities(ctx context.Context, deviceClientId string, capabilities sbi.DeviceCapabilitiesManifest, overrideOptions ...HTTPApiClientRequestEditorOptions) error {
	resp, err := self.client.PostApiV1ClientsClientIdCapabilities(ctx, deviceClientId, capabilities)
	if err != nil {
		return fmt.Errorf("failed to report capabilities: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("capabilities reporting failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (self *SbiHttpClient) SyncState(ctx context.Context, deviceClientId string, etag string, overrideOptions ...HTTPApiClientRequestEditorOptions) (desiredStates *sbi.UnsignedAppStateManifest, err error) {
    // Prepare parameters
    params := &sbi.GetApiV1ClientsClientIdDeploymentsParams{
        Accept: pointers.Ptr("application/vnd.margo.manifest.v1+json"),
    }
    
    // Only set If-None-Match if etag is not empty
    if etag != "" && etag != `""` { // Also check for quoted empty string
        params.IfNoneMatch = &etag
    }
    
    resp, err := self.client.GetApiV1ClientsClientIdDeployments(
        ctx,
        deviceClientId,
        params,
        overrideOptions...,
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse response first
    desiredStateResp, err := sbi.ParseGetApiV1ClientsClientIdDeploymentsResponse(resp)
    if err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    // Handle status codes according to OpenAPI spec
    switch resp.StatusCode {
    case 200:
        // OK - new data available
        if desiredStateResp.ApplicationvndMargoManifestV1JSON200 != nil {
            return desiredStateResp.ApplicationvndMargoManifestV1JSON200, nil
        }
        return nil, fmt.Errorf("unexpected response structure for status 200")
        
    case 304:
        // Not Modified - no new data
        return nil, nil
        
    case 406:
        // Not Acceptable - server cannot generate response matching Accept header
        return nil, fmt.Errorf("server cannot generate response matching Accept header")
        
    default:
        return nil, fmt.Errorf("unexpected status code returned by server: %d", resp.StatusCode)
    }
}
// SyncStateWithResponse retrieves the desired state manifest and returns the HTTP response for header access
func (self *SbiHttpClient) SyncStateWithResponse(ctx context.Context, deviceClientId string, etag string, overrideOptions ...HTTPApiClientRequestEditorOptions) (desiredStates *sbi.UnsignedAppStateManifest, response *http.Response, err error) {
    // Prepare parameters
    params := &sbi.GetApiV1ClientsClientIdDeploymentsParams{
        Accept: pointers.Ptr("application/vnd.margo.manifest.v1+json"),
    }
    
    // Only set If-None-Match if etag is not empty
    if etag != "" && etag != `""` {
        params.IfNoneMatch = &etag
    }
    
    resp, err := self.client.GetApiV1ClientsClientIdDeployments(
        ctx,
        deviceClientId,
        params,
        overrideOptions...,
    )
    if err != nil {
        return nil, nil, err
    }

    // DEBUG: Log the actual status code received
    //fmt.Printf("DEBUG: Received HTTP status code: %d\n", resp.StatusCode)
    
    // Check status code BEFORE parsing response
    // 304 Not Modified has no body, so don't try to parse it
    if resp.StatusCode == 304 {
        //fmt.Printf("DEBUG: Status is 304, returning early without parsing\n")
        return nil, resp, nil
    }

    //fmt.Printf("DEBUG: Status is NOT 304, proceeding to parse response\n")
    
    // Only parse response for status codes that have a body
    desiredStateResp, err := sbi.ParseGetApiV1ClientsClientIdDeploymentsResponse(resp)
    if err != nil {
        resp.Body.Close()
        fmt.Printf("DEBUG: Parse failed with error: %v\n", err)
        return nil, nil, fmt.Errorf("failed to parse response: %w", err)
    }

    // Handle status codes according to OpenAPI spec
    switch resp.StatusCode {
    case 200:
        // OK - new data available
        if desiredStateResp.ApplicationvndMargoManifestV1JSON200 != nil {
            return desiredStateResp.ApplicationvndMargoManifestV1JSON200, resp, nil
        }
        resp.Body.Close()
        return nil, nil, fmt.Errorf("unexpected response structure for status 200")
        
    case 406:
        // Not Acceptable
        resp.Body.Close()
        return nil, nil, fmt.Errorf("server cannot generate response matching Accept header")
        
    default:
        resp.Body.Close()
        return nil, nil, fmt.Errorf("unexpected status code returned by server: %d", resp.StatusCode)
    }
}



func (self *SbiHttpClient) ReportDeploymentStatus(ctx context.Context, deviceID, appID string, overallAppStatus sbi.DeploymentStatusManifestStatusState, components []sbi.ComponentStatus, deploymentErr error) error {
    appUUID, err := uuid.Parse(appID)
    if err != nil {
        return err
    }

    // Build error struct only if there's an actual error
    var errorStruct *struct {
        Code    *string `json:"code,omitempty"`
        Message *string `json:"message,omitempty"`
    }
    
    if deploymentErr != nil {
        errorStruct = &struct {
            Code    *string `json:"code,omitempty"`
            Message *string `json:"message,omitempty"`
        }{
            Code:    pointers.Ptr("DEPLOYMENT_ERROR"),
            Message: pointers.Ptr(deploymentErr.Error()),
        }
    }

    deploymentStatus := sbi.DeploymentStatusManifest{
        ApiVersion:   "margo.org",
        Kind:         "DeploymentStatus",
        Components:   components,
        DeploymentId: appUUID.String(),
        Status: struct {
            Error *struct {
                Code    *string "json:\"code,omitempty\""
                Message *string "json:\"message,omitempty\""
            } "json:\"error,omitempty\""
            State sbi.DeploymentStatusManifestStatusState "json:\"state\""
        }{
            Error: errorStruct,  //  nil for success, populated for errors

            State: overallAppStatus,
        },
    }

    resp, err := self.client.PostApiV1ClientsClientIdDeploymentDeploymentIdStatus(ctx, deviceID, appUUID.String(), deploymentStatus)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}

// Add to SbiHttpClient in sbi.go
func (self *SbiHttpClient) FetchDeploymentYAML(ctx context.Context, deviceClientId, deploymentId, digest string, overrideOptions ...HTTPApiClientRequestEditorOptions) (yamlContent []byte, err error) {
    params := &sbi.GetApiV1ClientsClientIdDeploymentsDeploymentIdDigestParams{}
    
    resp, err := self.client.GetApiV1ClientsClientIdDeploymentsDeploymentIdDigest(
        ctx,
        deviceClientId,
        deploymentId,
        digest,
        params,
        overrideOptions...,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to fetch deployment YAML: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("deployment fetch failed with status: %d", resp.StatusCode)
    }
    
    // Read YAML content
    yamlContent, err = io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read deployment YAML: %w", err)
    }
    
    // CRITICAL: Verify digest (Exact Bytes Rule)
    hash := sha256.Sum256(yamlContent)
    actualDigest := fmt.Sprintf("sha256:%x", hash)
    
    if actualDigest != digest {
        return nil, fmt.Errorf("deployment digest mismatch: expected %s, got %s", 
            digest, actualDigest)
    }
    
    return yamlContent, nil
}

func (self *SbiHttpClient) DownloadBundle(ctx context.Context, deviceClientId, digest string, overrideOptions ...HTTPApiClientRequestEditorOptions) (bundleData []byte, err error) {
    params := &sbi.GetApiV1ClientsClientIdBundlesDigestParams{}
    
    resp, err := self.client.GetApiV1ClientsClientIdBundlesDigest(
        ctx,
        deviceClientId,
        digest,
        params,
        overrideOptions...,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to download bundle: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("bundle download failed with status: %d", resp.StatusCode)
    }
    
    // Read bundle data
    bundleData, err = io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read bundle: %w", err)
    }
    
    // Verify digest (Exact Bytes Rule)
    hash := sha256.Sum256(bundleData)
    actualDigest := fmt.Sprintf("sha256:%x", hash)
    
    if actualDigest != digest {
        return nil, fmt.Errorf("bundle digest mismatch: expected %s, got %s", 
            digest, actualDigest)
    }
    
    return bundleData, nil
}

