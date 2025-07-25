// poc/device/agent/agent.go
package agent

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

// DeviceAgent represents the main device agent
type DeviceAgent struct {
	config     *AgentConfig
	device     *Device
	server     *http.Server
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// AgentConfig holds configuration for the device agent
type AgentConfig struct {
	DeviceID          string        `json:"device_id"`
	ServerURL         string        `json:"server_url"`
	CertPath          string        `json:"cert_path"`
	KeyPath           string        `json:"key_path"`
	CredsPath         string        `json:"creds_path"`
	ListenPort        int           `json:"listen_port"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

// DeviceCredentials represents stored device credentials
type DeviceCredentials struct {
	DeviceID    string    `json:"device_id"`
	Certificate []byte    `json:"certificate"`
	PrivateKey  []byte    `json:"private_key"`
	ServerURL   string    `json:"server_url"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Device represents the device instance
type Device struct {
	ID           string
	Credentials  *DeviceCredentials
	Status       DeviceStatus
	Capabilities []string
	LastSeen     time.Time
}

// DeviceStatus represents device operational status
type DeviceStatus string

const (
	StatusOffline     DeviceStatus = "offline"
	StatusOnline      DeviceStatus = "online"
	StatusRegistering DeviceStatus = "registering"
	StatusError       DeviceStatus = "error"
)

// SelfCertificates represents device's own certificates
type SelfCertificates struct {
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
	CSR         []byte `json:"csr"`
}

// NewDeviceAgent creates a new device agent instance
func NewDeviceAgent(config *AgentConfig) *DeviceAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &DeviceAgent{
		config:     config,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Start initializes and starts the device agent
func (da *DeviceAgent) Start() error {
	log.Println("Starting device agent...")
	// 1. Check if device is already registered
	if !da.isAlreadyRegistered() {
		log.Println("Device not registered, starting registration process...")

		// Generate or read self certificates
		selfCerts, err := da.readOrGenerateSelfCerts()
		if err != nil {
			return fmt.Errorf("failed to get self certificates: %w", err)
		}
		// Register device with server
		creds, err := da.registerDevice(selfCerts)
		if err != nil {
			return fmt.Errorf("failed to register device: %w", err)
		}
		// Store credentials
		if err := da.storeCreds(creds); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}
	}
	// 2. Read stored credentials
	creds, err := da.readCreds()
	if err != nil {
		return fmt.Errorf("failed to read credentials: %w", err)
	}
	// 3. Initialize device
	device, err := da.newDevice(creds)
	if err != nil {
		return fmt.Errorf("failed to initialize device: %w", err)
	}
	da.device = device
	// 4. Start HTTP server
	if err := da.startServer(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	// 5. Start background services
	go da.startHeartbeat()
	go da.startStateMonitoring()
	log.Printf("Device agent started successfully. Device ID: %s", da.device.ID)
	return nil
}

// Stop gracefully shuts down the device agent
func (da *DeviceAgent) Stop() error {
	log.Println("Stopping device agent...")

	da.cancelFunc()

	if da.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return da.server.Shutdown(ctx)
	}

	return nil
}

// isAlreadyRegistered checks if device credentials exist on disk
func (da *DeviceAgent) isAlreadyRegistered() bool {
	_, err := os.Stat(da.config.CredsPath)
	return err == nil
}

// readOrGenerateSelfCerts reads existing certificates or generates new ones
func (da *DeviceAgent) readOrGenerateSelfCerts() (*SelfCertificates, error) {
	// Check if certificates already exist
	if da.certificatesExist() {
		return da.readSelfCerts()
	}

	// Generate new certificates
	return da.generateSelfCerts()
}

// certificatesExist checks if certificate files exist on disk
func (da *DeviceAgent) certificatesExist() bool {
	_, certErr := os.Stat(da.config.CertPath)
	_, keyErr := os.Stat(da.config.KeyPath)
	return certErr == nil && keyErr == nil
}

// readSelfCerts reads existing certificates from disk
func (da *DeviceAgent) readSelfCerts() (*SelfCertificates, error) {
	certPEM, err := ioutil.ReadFile(da.config.CertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}
	keyPEM, err := ioutil.ReadFile(da.config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	return &SelfCertificates{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	}, nil
}

// generateSelfCerts generates new RSA key pair and CSR
func (da *DeviceAgent) generateSelfCerts() (*SelfCertificates, error) {
	log.Println("Generating new key pair and CSR...")
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	// Create CSR template
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   da.config.DeviceID,
			Organization: []string{"IoT Device"},
		},
		DNSNames: []string{da.config.DeviceID},
	}

	// Create CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}
	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	// Encode CSR to PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	// Save to disk
	if err := da.saveKeyPair(privateKeyPEM, csrPEM); err != nil {
		return nil, fmt.Errorf("failed to save key pair: %w", err)
	}
	return &SelfCertificates{
		PrivateKey: privateKeyPEM,
		CSR:        csrPEM,
	}, nil
}

// saveKeyPair saves private key and CSR to disk
func (da *DeviceAgent) saveKeyPair(privateKeyPEM, csrPEM []byte) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(da.config.KeyPath), 0700); err != nil {
		return err
	}
	// Save private key
	if err := ioutil.WriteFile(da.config.KeyPath, privateKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}
	// Save CSR (temporary, for debugging)
	csrPath := da.config.KeyPath + ".csr"
	if err := ioutil.WriteFile(csrPath, csrPEM, 0644); err != nil {
		return fmt.Errorf("failed to save CSR: %w", err)
	}
	return nil
}

// registerDevice registers the device with the onboarding server
func (da *DeviceAgent) registerDevice(selfCerts *SelfCertificates) (*DeviceCredentials, error) {
	log.Println("Registering device with server...")
	// Prepare registration request
	regReq := map[string]interface{}{
		"device_id": da.config.DeviceID,
		"csr":       string(selfCerts.CSR),
		"metadata": map[string]interface{}{
			"agent_version": "1.0.0",
			"capabilities":  []string{"telemetry", "remote_control", "firmware_update"},
		},
	}
	// Send registration request
	// This is a simplified implementation - in reality, you'd use proper HTTP client with retries
	resp, err := da.sendRegistrationRequest(regReq)
	if err != nil {
		return nil, fmt.Errorf("registration request failed: %w", err)
	}

	// Parse response and create credentials
	creds := &DeviceCredentials{
		DeviceID:    da.config.DeviceID,
		Certificate: []byte(resp["certificate"].(string)),
		PrivateKey:  selfCerts.PrivateKey,
		ServerURL:   da.config.ServerURL,
		IssuedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(365 * 24 * time.Hour), // 1 year
	}
	return creds, nil
}

// sendRegistrationRequest sends HTTP request to registration server
func (da *DeviceAgent) sendRegistrationRequest(regReq map[string]interface{}) (map[string]interface{}, error) {
	// This is a stub implementation
	// In reality, you'd implement proper HTTP client with authentication, retries, etc.

	log.Printf("Sending registration request to %s", da.config.ServerURL)

	// Simulate successful registration
	return map[string]interface{}{
		"status":      "success",
		"certificate": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
		"device_id":   da.config.DeviceID,
	}, nil
}

// storeCreds saves device credentials to disk
func (da *DeviceAgent) storeCreds(creds *DeviceCredentials) error {
	log.Println("Storing device credentials...")
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(da.config.CredsPath), 0700); err != nil {
		return err
	}
	// Marshal credentials to JSON
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}
	// Write to file
	if err := ioutil.WriteFile(da.config.CredsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}
	// Also save certificate separately
	if err := ioutil.WriteFile(da.config.CertPath, creds.Certificate, 0644); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}
	return nil
}

// readCreds reads device credentials from disk
func (da *DeviceAgent) readCreds() (*DeviceCredentials, error) {
	data, err := ioutil.ReadFile(da.config.CredsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}
	var creds DeviceCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}
	return &creds, nil
}

// newDevice creates a new device instance with credentials
func (da *DeviceAgent) newDevice(creds *DeviceCredentials) (*Device, error) {
	device := &Device{
		ID:           creds.DeviceID,
		Credentials:  creds,
		Status:       StatusOnline,
		Capabilities: []string{"telemetry", "remote_control", "firmware_update"},
		LastSeen:     time.Now(),
	}
	return device, nil
}

// startServer starts the HTTP server for device management
func (da *DeviceAgent) startServer() error {
	router := mux.NewRouter()

	// Device management endpoints
	router.HandleFunc("/status", da.handleStatus).Methods("GET")
	router.HandleFunc("/capabilities", da.handleCapabilities).Methods("GET")
	router.HandleFunc("/command", da.handleCommand).Methods("POST")

	da.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", da.config.ListenPort),
		Handler: router,
	}
	go func() {
		log.Printf("Starting HTTP server on port %d", da.config.ListenPort)
		if err := da.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()
	return nil
}

// startHeartbeat starts the heartbeat routine
func (da *DeviceAgent) startHeartbeat() {
	ticker := time.NewTicker(da.config.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-da.ctx.Done():
			return
		case <-ticker.C:
			da.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends heartbeat to server
func (da *DeviceAgent) sendHeartbeat() {
	log.Println("Sending heartbeat...")
	da.device.LastSeen = time.Now()
	// Implementation would send actual heartbeat to server
}

// startStateMonitoring starts device state monitoring
func (da *DeviceAgent) startStateMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-da.ctx.Done():
			return
		case <-ticker.C:
			da.monitorDeviceState()
		}
	}
}

// monitorDeviceState monitors and reports device state
func (da *DeviceAgent) monitorDeviceState() {
	// Implementation would monitor device health, resources, etc.
	log.Println("Monitoring device state...")
}

// HTTP Handlers

func (da *DeviceAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"device_id": da.device.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (da *DeviceAgent) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(da.device)
}

func (da *DeviceAgent) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"capabilities": da.device.Capabilities,
		"device_id":    da.device.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (da *DeviceAgent) handleCommand(w http.ResponseWriter, r *http.Request) {
	var command map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
		http.Error(w, "Invalid command format", http.StatusBadRequest)
		return
	}
	// Process command
	result := da.processCommand(command)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// processCommand processes incoming commands
func (da *DeviceAgent) processCommand(command map[string]interface{}) map[string]interface{} {
	cmdType, ok := command["type"].(string)
	if !ok {
		return map[string]interface{}{
			"status": "error",
			"error":  "missing command type",
		}
	}
	switch cmdType {
	case "restart":
		return map[string]interface{}{
			"status":  "success",
			"message": "restart command received",
		}
	case "update_config":
		return map[string]interface{}{
			"status":  "success",
			"message": "config update command received",
		}
	default:
		return map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("unknown command type: %s", cmdType),
		}
	}
}

// Main function
func main() {
	config := &AgentConfig{
		DeviceID:          "device-001",
		ServerURL:         "https://onboarding.example.com",
		CertPath:          "/etc/device/cert.pem",
		KeyPath:           "/etc/device/key.pem",
		CredsPath:         "/etc/device/credentials.json",
		ListenPort:        8080,
		HeartbeatInterval: 30 * time.Second,
	}
	agent := NewDeviceAgent(config)

	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start device agent: %v", err)
	}
	// Wait for interrupt signal
	select {}
}
