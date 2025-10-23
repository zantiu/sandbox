package crypto

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/lestrrat-go/htmsig/component"
	htmsighttp "github.com/lestrrat-go/htmsig/http"
)

type HTTPVerifier interface {
	VerifyRequest(ctx context.Context, req *http.Request) error
	VerifyResponse(ctx context.Context, resp *http.ResponseWriter) error
}

type HTMPayloadVerifier struct {
	// keep original string for debugging
	publicKeyPEM string
	verifier     *htmsighttp.Verifier
}

func NewVerifier(publicKey string, isPubKeyBase64 bool) (*HTMPayloadVerifier, error) {
	if isPubKeyBase64 {
		der, err := base64.StdEncoding.DecodeString(publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 public key, %s", err.Error())
		}
		publicKey = string(der)
	}

	// try to parse PEM public key to supply a parsed key to the resolver
	block, _ := pem.Decode([]byte(publicKey))
	var parsedKey any
	var err error
	if block != nil {
		parsedKey, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key PEM: %w", err)
		}
	} else {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}

	resolver := htmsighttp.StaticKeyResolver(parsedKey)
	verifier := htmsighttp.NewVerifier(resolver, htmsighttp.WithComponents(
		component.Method(),
		component.TargetURI(),
		component.Authority(),
	))
	return &HTMPayloadVerifier{
		publicKeyPEM: publicKey,
		verifier:     verifier,
	}, nil
}

func (self *HTMPayloadVerifier) VerifyRequest(ctx context.Context, req *http.Request) error {
	return self.verifier.VerifyRequest(ctx, req)
}

func (self *HTMPayloadVerifier) VerifyResponse(ctx context.Context, resp *http.ResponseWriter) error {
	return fmt.Errorf("response verifier is not implemented")
}
