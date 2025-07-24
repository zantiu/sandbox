package agent

import (
	"fmt"
	"log"

	"github.com/margo/dev-repo/shared-lib/certs/pki"
)

type DeviceAgent struct{}

func NewDeviceAgent(certPath string) (*DeviceAgent, error) {
	// load the cert from file, if it exists
	return nil, nil
}

func main() {
	fmt.Println("I'm a device agent!")
	// 1. check if the onboarding cert is already present on the disk
	// if so, then read it, and start the server
	// 2. if the onboarding cert is not present on the disk,
	// then we need to onboard the device
	//    a. Check if private key pair is present on the disk
	//		b. else generate the private key pair and store on disk
	//    c. finally hit the wfm api to onboard the device
}

func (main *DeviceAgent) GeneratePrivateKey() {
	// create pki key and cert
	// send the csr containing the public key to server
	// the server will send back the signed certificate
	// store
}

func (main *DeviceAgent) OnboardWithWFM() {
	// create pki key and cert
	// send the csr containing the public key to server
	// the server will send back the signed certificate
	// store
}

func onboardWithWFM() {
	// create pki key and cert
	// send the csr containing the public key to server
	// the server will send back the signed certificate
	// store this certificate in the file
	// use the pki package to acheive

	// Generate a new private key and certificate signing request
	privateKey, err := pki.GeneratePrivateKey()
	if err != nil {
		log.Printf("Failed to generate private key: %v", err)
		return
	}

	// Create CSR with device information
	csr, err := pki.CreateCSR(privateKey, "device-id")
	if err != nil {
		log.Printf("Failed to create CSR: %v", err)
		return
	}

	// Send CSR to WFM server for signing
	signedCert, err := sendCSRToServer(csr)
	if err != nil {
		log.Printf("Failed to get signed certificate from server: %v", err)
		return
	}

	// Store the private key and signed certificate
	err = storeCertificate(privateKey, signedCert)
	if err != nil {
		log.Printf("Failed to store certificate: %v", err)
		return
	}

	log.Printf("Successfully onboarded with WFM")

}

func deboardFromWFM() {}

func shutdown() {}

func reset() {}

func restart() {}
