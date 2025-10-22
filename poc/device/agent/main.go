// agent.go - Updated version
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"github.com/margo/dev-repo/poc/device/agent/database"
	"github.com/margo/dev-repo/poc/device/agent/types"
	wfm "github.com/margo/dev-repo/poc/wfm/cli"
	"github.com/margo/dev-repo/shared-lib/crypto"
	"github.com/margo/dev-repo/shared-lib/workloads"
	"go.uber.org/zap"
)

type Agent struct {
	log            *zap.SugaredLogger
	auth           *DeviceClientSettings
	config         types.Config
	database       database.DatabaseIfc
	syncer         StateSyncerIfc
	deployer       DeploymentManagerIfc
	monitor        DeploymentMonitorIfc
	statusReporter StatusReporterIfc
}

func NewAgent(configPath string) (*Agent, error) {
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Load configuration
	cfg, err := types.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Create database
	db := database.NewDatabase("data/")

	// Prepare request editors (e.g., request signer) for WFM client
	var requestEditors []wfm.HTTPApiClientOptions

	// Create WFM client using configured URL
	wfmUrl := cfg.Wfm.SbiURL
	if wfmUrl == "" {
		return nil, fmt.Errorf("wfm.sbiUrl is empty in configuration")
	}

	hasRequestSigningKey := false
	// If request signer plugin enabled, create signer and add as RequestEditorFn
	if cfg.Wfm.ClientPlugins.RequestSigner != nil && cfg.Wfm.ClientPlugins.RequestSigner.Enabled {
		if cfg.Wfm.ClientPlugins.RequestSigner.KeyRef == nil {
			return nil, fmt.Errorf("request signer enabled but no keyRef provided in configuration")
		}
		// read private key from file
		signer, err := crypto.NewSignerFromFile(
			cfg.Wfm.ClientPlugins.RequestSigner.KeyRef.Path,
			cfg.Wfm.ClientPlugins.RequestSigner.SignatureAlgo,
			cfg.Wfm.ClientPlugins.RequestSigner.HashAlgo,
			cfg.Wfm.ClientPlugins.RequestSigner.SignatureFormat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create request signer: %w", err)
		}

		hasRequestSigningKey = true
		// adapter to the generated client's RequestEditorFn signature
		requestEditors = append(requestEditors, func(ctx context.Context, req *http.Request) error {
			return signer.SignRequest(ctx, req)
		})
	}

	wfmClient, err := wfm.NewSbiHTTPClient(wfmUrl, requestEditors...)
	if err != nil {
		return nil, fmt.Errorf("failed to create WFM client: %w", err)
	}

	opts := []Option{}
	var helmClient *workloads.HelmClient
	var composeClient *workloads.DockerComposeCliClient
	for _, runtime := range cfg.Runtimes {
		if runtime.Kubernetes != nil {
			// Create Helm client
			helmClient, err = workloads.NewHelmClient(runtime.Kubernetes.KubeconfigPath)
			if err != nil {
				return nil, err
			}
			opts = append(opts, WithEnableHelmDeployment())
		}

		if runtime.Docker != nil {
			// Create docker compose client
			composeClient, err = workloads.NewDockerComposeCliClient(workloads.DockerConnectivityParams{
				ViaSocket: &workloads.DockerConnectionViaSocket{
					SocketPath: runtime.Docker.Url,
				},
			}, "data/composeFiles")
			if err != nil {
				return nil, err
			}
			opts = append(opts, WithEnableComposeDeployment())
		}
	}
	if helmClient == nil && composeClient == nil {
		return nil, fmt.Errorf("neither kubernetes nor docker runtime objects were able to be attached, please check info if you have misplaced their settings")
	}

	opts = append(opts, WithDeviceRootIdentity(findDeviceRootIdentity(*cfg, log)))

	var deviceSettings *DeviceClientSettings
	deviceSettings, err = NewDeviceSettings(wfmClient, db, log, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize device settings: %w", err)
	}
	isOnboarded, err := deviceSettings.IsOnboarded()
	if err != nil {
		log.Errorw("failed to check onboarding status", "error", err)
		return nil, err
	}

	if !isOnboarded {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		deviceId, err := deviceSettings.OnboardWithRetries(ctx, 10)
		if err != nil {
			log.Errorw("device onboarding failed", "error", err)
			return nil, fmt.Errorf("failed to onboard the device, %s", err.Error())
		}
		log.Infow("device onboarded", "deviceId", deviceId)
	}

	// Determine signature/certificate availability from deviceSettings (adapt to new attestation model)
	hasValidDeviceCertificate := false
	if deviceSettings != nil {
		if deviceSettings.deviceRootIdentity.HasCertificateReference() {
			hasValidDeviceCertificate = true
		}
		if deviceSettings.deviceRootIdentity.IdentityType == "Random" && deviceSettings.deviceRootIdentity.Attestation.Random != nil && deviceSettings.deviceRootIdentity.Attestation.Random.Value != "" {
			hasValidDeviceCertificate = true
		}
	}

	log.Infow("device details",
		"deviceId", deviceSettings.deviceClientId,
		"deviceSignatureType", deviceSettings.deviceRootIdentity.IdentityType,
		"hasValidDeviceCertificate", hasValidDeviceCertificate,
		"canSignRequests", hasRequestSigningKey,
		"canDeployHelm", deviceSettings.canDeployHelm,
		"canDeployCompose", deviceSettings.canDeployCompose,
		"isAuthEnabled", deviceSettings.authEnabled,
		"hasClientId", len(deviceSettings.oauthClientId) != 0,
		"hasClientSecret", len(deviceSettings.oAuthClientSecret) != 0,
		"hasTokenUrl", len(deviceSettings.oauthTokenUrl) != 0,
		"tokenBasedAuthDetails", (len(deviceSettings.oauthClientId) != 0) && (len(deviceSettings.oAuthClientSecret) != 0) && (len(deviceSettings.oauthTokenUrl) != 0),
	)

	// Create components
	deployer := NewDeploymentManager(db, helmClient, composeClient, log)
	monitor := NewDeploymentMonitor(db, helmClient, composeClient, log)
	syncer := NewStateSyncer(db, wfmClient, deviceSettings.deviceClientId, cfg.StateSeeking.Interval, log)
	statusReporter := NewStatusReporter(db, wfmClient, deviceSettings.deviceClientId, log)

	return &Agent{
		database:       db,
		syncer:         syncer,
		deployer:       deployer,
		monitor:        monitor,
		auth:           deviceSettings,
		statusReporter: statusReporter,
		log:            log,
		config:         *cfg,
	}, nil
}

func (a *Agent) Start() error {
	a.log.Info("Starting Agent")

	var deviceId string
	var err error

	// 1. Onboard device
	deviceSettings, _ := a.database.GetDeviceSettings()
	deviceId = deviceSettings.DeviceClientId

	// 2. Report capabilities
	capabilities, err := types.LoadCapabilities(a.config.Capabilities.ReadFromFile)
	if err != nil {
		a.log.Errorw(
			"failed to load the capabilities file, please resolve the issue as the capabilities will not be reported until next restart",
			"err",
			err.Error(),
		)
	} else {
		capabilities.Properties.Id = deviceId
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		a.auth.ReportCapabilities(ctx, *capabilities)
		cancel()
	}

	// 3. Start all components
	a.statusReporter.Start()
	a.deployer.Start()
	a.monitor.Start()
	a.syncer.Start()

	hasCfgPubCert := false
	if a.config.DeviceRootIdentity.HasCertificateReference() {
		hasCfgPubCert = true
	}

	a.log.Infow("Agent started successfully",
		"capabilitiesFile", a.config.Capabilities.ReadFromFile,
		"hasDeviceSignature", hasCfgPubCert,
		"stateSeekingInterval", a.config.StateSeeking.Interval,
		"sbiUrl", a.config.Wfm.SbiURL,
	)
	return nil
}

func (a *Agent) Stop() error {
	a.log.Info("Stopping Agent")

	a.syncer.Stop()
	a.deployer.Stop()
	a.monitor.Stop()
	a.statusReporter.Stop()
	a.database.TriggerDataPersist()

	a.log.Info("Agent stopped")
	return nil
}

func findDeviceRootIdentity(cfg types.Config, logger *zap.SugaredLogger) types.DeviceRootIdentity {
	return cfg.DeviceRootIdentity
}

func main() {
	// Define command-line flags
	configPath := flag.String(
		"config",
		"poc/device/agent/config/config.yaml", // default value
		"Path to the YAML configuration file for the Margo device agent",
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nMargo Device Agent\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if configPath == nil {
		log.Fatal("--config is mandatory command line argument")
	}

	agent, err := NewAgent(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := agent.Start(); err != nil {
		log.Fatal(err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	agent.Stop()
}
