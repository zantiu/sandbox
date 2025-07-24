package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	. "github.com/margo/dev-repo/non-standard/generatedCode/wfm/sbi"
)

// ValidationError represents a validation error with field context
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Regular expressions for validation
var (
	deviceNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	namespaceRegex  = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	labelKeyRegex   = regexp.MustCompile(`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\/)?[a-z0-9A-Z]([-a-z0-9A-Z_.]*[a-z0-9A-Z])?$`)
	labelValueRegex = regexp.MustCompile(`^[a-z0-9A-Z]([-a-z0-9A-Z_.]*[a-z0-9A-Z])?$`)
)

// ValidateDeviceOnboardingRequest validates a DeviceOnboardingRequest
func ValidateDeviceOnboardingRequest(req *DeviceOnboardingRequest) ValidationErrors {
	var errors ValidationErrors

	if req == nil {
		return ValidationErrors{{Field: "request", Message: "request cannot be nil"}}
	}

	// Validate required fields
	if req.ApiVersion == "" {
		errors = append(errors, ValidationError{Field: "apiVersion", Message: "apiVersion is required"})
	} else if req.ApiVersion != "v1" {
		errors = append(errors, ValidationError{Field: "apiVersion", Message: "apiVersion must be 'v1'"})
	}

	if req.Kind == "" {
		errors = append(errors, ValidationError{Field: "kind", Message: "kind is required"})
	} else if req.Kind != "DeviceOnboardingRequest" {
		errors = append(errors, ValidationError{Field: "kind", Message: "kind must be 'DeviceOnboardingRequest'"})
	}

	// Validate metadata
	metadataErrors := validateDeviceOnboardingRequestMetadata(&req.Metadata)
	for _, err := range metadataErrors {
		err.Field = "metadata." + err.Field
		errors = append(errors, err)
	}

	// Validate spec
	specErrors := validateDeviceOnboardingRequestSpec(&req.Spec)
	for _, err := range specErrors {
		err.Field = "spec." + err.Field
		errors = append(errors, err)
	}

	return errors
}

// validateDeviceOnboardingRequestMetadata validates the metadata section
func validateDeviceOnboardingRequestMetadata(metadata *DeviceOnboardingRequest_Metadata) ValidationErrors {
	var errors ValidationErrors

	if metadata == nil {
		return ValidationErrors{{Field: "metadata", Message: "metadata is required"}}
	}

	// Validate name
	if metadata.Name != nil {
		if *metadata.Name == "" {
			errors = append(errors, ValidationError{Field: "name", Message: "name cannot be empty"})
		} else if len(*metadata.Name) > 253 {
			errors = append(errors, ValidationError{Field: "name", Message: "name cannot exceed 253 characters"})
		} else if !deviceNameRegex.MatchString(*metadata.Name) {
			errors = append(errors, ValidationError{Field: "name", Message: "name must be a valid DNS-1123 subdomain"})
		}
	}

	// Validate namespace
	if metadata.Namespace != nil {
		if *metadata.Namespace == "" {
			errors = append(errors, ValidationError{Field: "namespace", Message: "namespace cannot be empty"})
		} else if len(*metadata.Namespace) > 63 {
			errors = append(errors, ValidationError{Field: "namespace", Message: "namespace cannot exceed 63 characters"})
		} else if !namespaceRegex.MatchString(*metadata.Namespace) {
			errors = append(errors, ValidationError{Field: "namespace", Message: "namespace must be a valid DNS-1123 label"})
		}
	}

	// Validate labels
	if metadata.Labels != nil {
		labelErrors := validateLabels(*metadata.Labels, "labels")
		errors = append(errors, labelErrors...)
	}

	// Validate annotations
	if metadata.Annotations != nil {
		annotationErrors := validateAnnotations(*metadata.Annotations, "annotations")
		errors = append(errors, annotationErrors...)
	}

	return errors
}

// validateDeviceOnboardingRequestSpec validates the spec section
func validateDeviceOnboardingRequestSpec(spec *struct {
	Protocol *AuthProtocol `json:"protocol,omitempty"`
}) ValidationErrors {
	var errors ValidationErrors

	if spec == nil {
		return ValidationErrors{{Field: "spec", Message: "spec is required"}}
	}

	// Validate protocol
	if spec.Protocol != nil {
		protocolErrors := validateAuthProtocol(spec.Protocol)
		for _, err := range protocolErrors {
			err.Field = "protocol." + err.Field
			errors = append(errors, err)
		}
	}

	return errors
}

// ValidateDevice validates a Device struct
func ValidateDevice(device *Device) ValidationErrors {
	var errors ValidationErrors

	if device == nil {
		return ValidationErrors{{Field: "device", Message: "device cannot be nil"}}
	}

	// Validate required fields
	if device.ApiVersion == "" {
		errors = append(errors, ValidationError{Field: "apiVersion", Message: "apiVersion is required"})
	} else if device.ApiVersion != "v1" {
		errors = append(errors, ValidationError{Field: "apiVersion", Message: "apiVersion must be 'v1'"})
	}

	if device.Kind == "" {
		errors = append(errors, ValidationError{Field: "kind", Message: "kind is required"})
	} else if device.Kind != "Device" {
		errors = append(errors, ValidationError{Field: "kind", Message: "kind must be 'Device'"})
	}

	// Validate metadata
	metadataErrors := validateDeviceMetadata(&device.Metadata)
	for _, err := range metadataErrors {
		err.Field = "metadata." + err.Field
		errors = append(errors, err)
	}

	// Validate spec
	if device.Spec.Protocol != nil {
		protocolErrors := validateAuthProtocol(device.Spec.Protocol)
		for _, err := range protocolErrors {
			err.Field = "spec.protocol." + err.Field
			errors = append(errors, err)
		}
	}

	// Validate status
	if device.Status != nil {
		statusErrors := validateDeviceStatus(device.Status)
		for _, err := range statusErrors {
			err.Field = "status." + err.Field
			errors = append(errors, err)
		}
	}

	// Validate recent operation
	if device.RecentOperation != nil {
		recentOpErrors := validateDeviceRecentOperation(device.RecentOperation)
		for _, err := range recentOpErrors {
			err.Field = "recentOperation." + err.Field
			errors = append(errors, err)
		}
	}

	return errors
}

// validateDeviceMetadata validates Device metadata
func validateDeviceMetadata(metadata *Device_Metadata) ValidationErrors {
	var errors ValidationErrors

	// Validate name
	if metadata.Name != nil {
		if *metadata.Name == "" {
			errors = append(errors, ValidationError{Field: "name", Message: "name cannot be empty"})
		} else if len(*metadata.Name) > 253 {
			errors = append(errors, ValidationError{Field: "name", Message: "name cannot exceed 253 characters"})
		} else if !deviceNameRegex.MatchString(*metadata.Name) {
			errors = append(errors, ValidationError{Field: "name", Message: "name must be a valid DNS-1123 subdomain"})
		}
	}

	// Validate namespace
	if metadata.Namespace != nil {
		if *metadata.Namespace == "" {
			errors = append(errors, ValidationError{Field: "namespace", Message: "namespace cannot be empty"})
		} else if len(*metadata.Namespace) > 63 {
			errors = append(errors, ValidationError{Field: "namespace", Message: "namespace cannot exceed 63 characters"})
		} else if !namespaceRegex.MatchString(*metadata.Namespace) {
			errors = append(errors, ValidationError{Field: "namespace", Message: "namespace must be a valid DNS-1123 label"})
		}
	}

	// Validate ID format (if present, should be server-generated)
	if metadata.Id != nil && *metadata.Id == "" {
		errors = append(errors, ValidationError{Field: "id", Message: "id cannot be empty if provided"})
	}

	// Validate creation timestamp (should not be in the future)
	if metadata.CreationTimestamp != nil && metadata.CreationTimestamp.After(time.Now()) {
		errors = append(errors, ValidationError{Field: "creationTimestamp", Message: "creationTimestamp cannot be in the future"})
	}

	// Validate labels
	if metadata.Labels != nil {
		labelErrors := validateLabels(*metadata.Labels, "labels")
		errors = append(errors, labelErrors...)
	}

	// Validate annotations
	if metadata.Annotations != nil {
		annotationErrors := validateAnnotations(*metadata.Annotations, "annotations")
		errors = append(errors, annotationErrors...)
	}

	return errors
}

// validateAuthProtocol validates AuthProtocol
func validateAuthProtocol(protocol *AuthProtocol) ValidationErrors {
	var errors ValidationErrors

	if protocol == nil {
		return ValidationErrors{{Field: "protocol", Message: "protocol cannot be nil"}}
	}

	// Validate type
	switch protocol.Type {
	case AuthProtocolTypeFIDO, AuthProtocolTypePKI:
		// Valid types
	case "":
		errors = append(errors, ValidationError{Field: "type", Message: "type is required"})
	default:
		errors = append(errors, ValidationError{Field: "type", Message: "invalid protocol type"})
	}

	// Validate version format
	if protocol.Version != nil && *protocol.Version == "" {
		errors = append(errors, ValidationError{Field: "version", Message: "version cannot be empty if provided"})
	}

	// Validate parameters based on protocol type
	if protocol.Parameters != nil {
		paramErrors := validateAuthProtocolParameters(protocol.Parameters, protocol.Type)
		for _, err := range paramErrors {
			err.Field = "parameters." + err.Field
			errors = append(errors, err)
		}
	}

	return errors
}

// validateAuthProtocolParameters validates protocol-specific parameters
func validateAuthProtocolParameters(params *AuthProtocol_Parameters, protocolType AuthProtocolType) ValidationErrors {
	var errors ValidationErrors

	switch protocolType {
	case AuthProtocolTypePKI:
		pkiParams, err := params.AsPKIParameters()
		if err != nil {
			errors = append(errors, ValidationError{Field: "pki", Message: "invalid PKI parameters format"})
		} else {
			pkiErrors := validatePKIParameters(&pkiParams)
			errors = append(errors, pkiErrors...)
		}
	case AuthProtocolTypeFIDO:
		fidoParams, err := params.AsFIDOParameters()
		if err != nil {
			errors = append(errors, ValidationError{Field: "fido", Message: "invalid FIDO parameters format"})
		} else {
			fidoErrors := validateFIDOParameters(&fidoParams)
			errors = append(errors, fidoErrors...)
		}
	}

	return errors
}

// validatePKIParameters validates PKI-specific parameters
func validatePKIParameters(params *PKIParameters) ValidationErrors {
	var errors ValidationErrors

	// Validate algorithm
	if params.Algorithm != nil {
		switch *params.Algorithm {
		case RSA, ECDSA:
			// Valid algorithms
		default:
			errors = append(errors, ValidationError{Field: "algorithm", Message: "invalid algorithm"})
		}
	}

	// Validate key size
	if params.KeySize != nil {
		switch *params.KeySize {
		case N2048, N4096:
			// Valid key sizes
		default:
			errors = append(errors, ValidationError{Field: "keySize", Message: "invalid key size"})
		}
	}

	// Validate certificate authority
	if params.CertificateAuthority != nil && *params.CertificateAuthority == "" {
		errors = append(errors, ValidationError{Field: "certificateAuthority", Message: "certificateAuthority cannot be empty if provided"})
	}

	// Validate CSR
	if params.Csr != nil && len(*params.Csr) == 0 {
		errors = append(errors, ValidationError{Field: "csr", Message: "csr cannot be empty if provided"})
	}

	return errors
}

// validateFIDOParameters validates FIDO-specific parameters
func validateFIDOParameters(params *FIDOParameters) ValidationErrors {
	var errors ValidationErrors

	// Validate AppId format (should be a valid URL)
	if params.AppId != nil && *params.AppId == "" {
		errors = append(errors, ValidationError{Field: "appId", Message: "appId cannot be empty if provided"})
	}

	// Validate challenge
	if params.Challenge != nil && len(*params.Challenge) == 0 {
		errors = append(errors, ValidationError{Field: "challenge", Message: "challenge cannot be empty if provided"})
	}

	// Validate key handle
	if params.KeyHandle != nil && len(*params.KeyHandle) == 0 {
		errors = append(errors, ValidationError{Field: "keyHandle", Message: "keyHandle cannot be empty if provided"})
	}

	return errors
}

// validateDeviceStatus validates device status
func validateDeviceStatus(status *DeviceStatus) ValidationErrors {
	var errors ValidationErrors

	// Validate state
	if status.State != nil {
		switch *status.State {
		case BOOTING, DEBOARDED, ERROR, OFFLINE, ONBOARDED, ONLINE, UNDERMAINTENANCE:
			// Valid states
		default:
			errors = append(errors, ValidationError{Field: "state", Message: "invalid device state"})
		}
	}

	// Validate last update time (should not be in the future)
	if status.LastUpdateTime != nil && status.LastUpdateTime.After(time.Now()) {
		errors = append(errors, ValidationError{Field: "lastUpdateTime", Message: "lastUpdateTime cannot be in the future"})
	}

	return errors
}

// validateDeviceRecentOperation validates recent operation
func validateDeviceRecentOperation(op *DeviceRecentOperation) ValidationErrors {
	var errors ValidationErrors

	// Validate operation
	switch op.Op {
	case DEBOARD, ONBOARD, RESET, RESTART, SHUTDOWN, START, UPDATEFIRMWARE:
		// Valid operations
	default:
		errors = append(errors, ValidationError{Field: "op", Message: "invalid device operation"})
	}

	// Validate status
	switch op.Status {
	case CANCELLED, COMPLETED, FAILED, PENDING, REJECTED, SCHEDULED, TIMEOUT:
		// Valid statuses
	default:
		errors = append(errors, ValidationError{Field: "status", Message: "invalid operation status"})
	}

	return errors
}

// validateLabels validates Kubernetes-style labels
func validateLabels(labels map[string]string, fieldName string) ValidationErrors {
	var errors ValidationErrors

	for key, value := range labels {
		// Validate key
		if len(key) == 0 {
			errors = append(errors, ValidationError{Field: fieldName, Message: "label key cannot be empty"})
			continue
		}
		if len(key) > 253 {
			errors = append(errors, ValidationError{Field: fieldName, Message: fmt.Sprintf("label key '%s' exceeds 253 characters", key)})
			continue
		}
		if !labelKeyRegex.MatchString(key) {
			errors = append(errors, ValidationError{Field: fieldName, Message: fmt.Sprintf("label key '%s' is invalid", key)})
		}

		// Validate value
		if len(value) > 63 {
			errors = append(errors, ValidationError{Field: fieldName, Message: fmt.Sprintf("label value for key '%s' exceeds 63 characters", key)})
			continue
		}
		if value != "" && !labelValueRegex.MatchString(value) {
			errors = append(errors, ValidationError{Field: fieldName, Message: fmt.Sprintf("label value '%s' for key '%s' is invalid", value, key)})
		}
	}

	return errors
}

// validateAnnotations validates Kubernetes-style annotations
func validateAnnotations(annotations map[string]string, fieldName string) ValidationErrors {
	var errors ValidationErrors

	for key, value := range annotations {
		// Validate key (same rules as labels)
		if len(key) == 0 {
			errors = append(errors, ValidationError{Field: fieldName, Message: "annotation key cannot be empty"})
			continue
		}
		if len(key) > 253 {
			errors = append(errors, ValidationError{Field: fieldName, Message: fmt.Sprintf("annotation key '%s' exceeds 253 characters", key)})
			continue
		}
		if !labelKeyRegex.MatchString(key) {
			errors = append(errors, ValidationError{Field: fieldName, Message: fmt.Sprintf("annotation key '%s' is invalid", key)})
		}

		// Annotations can have longer values than labels
		if len(value) > 262144 { // 256KB limit
			errors = append(errors, ValidationError{Field: fieldName, Message: fmt.Sprintf("annotation value for key '%s' exceeds 256KB", key)})
		}
	}

	return errors
}
