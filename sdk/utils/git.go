package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitAuth holds authentication credentials for Git repository access.
//
// Note: For GitHub and similar services, use personal access tokens instead of passwords
// for enhanced security and to comply with authentication best practices.
type GitAuth struct {
	Username   string // Username for Git authentication
	Token      string // Personal access token or password for authentication
	CABundle   []byte // CA bundle (PEM encoded) for self-signed certificates
	ClientCert []byte // Client certificate (PEM encoded)
	ClientKey  []byte // Private key (PEM encoded) for client certificate
}

// ReadFromGit clones a Git repository to a temporary directory without authentication.
//
// Parameters:
//   - url: The HTTPS Git repository URL to clone (required, cannot be empty)
//   - branchOrTagName: The name of the branch to clone (required, cannot be empty)
//   - cloneToDir: Path to clone directory (optional, if not given a random path will be used inside /tmp directory)
//
// Returns:
//   - outputDirPath: The absolute path to the cloned repository directory
//   - err: An error if the clone operation fails
//
// Important Notes:
//   - The caller is responsible for cleaning up the returned directory path
//   - Only works with public repositories or repositories that allow anonymous access
//   - For private repositories, use ReadFromGitWithAuth with proper authentication
//   - Only HTTPS-based Git URLs are supported; SSH URLs are not supported
//
// Example:
//
//	outputDirPath, err := ReadFromGit("https://github.com/user/public-repo.git", "main")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer os.RemoveAll(outputDirPath) // Clean up when done
//
// See also: ReadFromGitWithAuth for repositories requiring authentication.
func ReadFromGit(url, branchOrTagName string, outputPath *string) (outputDirPath string, err error) {
	return ReadFromGitWithAuth(url, branchOrTagName, nil, outputPath)
}

// ReadFromGitWithAuth clones a Git repository to a temporary directory with optional authentication.
//
// This function clones the specified Git repository branch to a temporary directory and returns
// the path to the cloned repository. It supports HTTPS-based Git URLs with optional authentication
// but does not support SSH-based URLs.
//
// Parameters:
//   - url: The HTTPS Git repository URL to clone (required, cannot be empty)
//   - branchOrTagName: The name of the branch to clone (required, cannot be empty)
//   - auth: Optional authentication credentials for private repositories
//   - cloneToDir: Path to clone directory (optional, if not given a random path will be used inside /tmp directory)
//
// Returns:
//   - outputDirPath: The absolute path to the cloned repository directory
//   - err: An error if the clone operation fails
//
// Important Notes:
//   - The caller is responsible for cleaning up the returned directory path
//   - Only HTTP(S)-based Git URLs are supported; SSH URLs are not supported
//   - If outputPath var is not provided then the function creates a temporary directory with the pattern like "margo-git-{timestamp}"
//   - Progress information is written to os.Stdout during cloning
//   - The function performs a single-branch clone for efficiency
//
// Example:
//
//	outputDirPath, err := ReadFromGitWithAuth("https://github.com/user/repo.git", "main", nil, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer os.RemoveAll(outputDirPath) // Clean up when done
func ReadFromGitWithAuth(url, branchOrTagName string, auth *GitAuth, outputPath *string) (string, error) {
	// Validate URL
	if url == "" {
		return "", fmt.Errorf("git URL cannot be empty")
	}
	// Validate branch name
	if branchOrTagName == "" {
		return "", fmt.Errorf("git branchOrTagName cannot be empty")
	}

	// Extract repository name from URL for directory naming
	repoName := extractRepoName(url)
	if repoName == "" {
		return "", fmt.Errorf("invalid git URL format")
	}

	// Create temporary directory for cloning
	var tempDir string
	if outputPath != nil {
		tempDir = *outputPath
	} else {
		tempDir = filepath.Join(os.TempDir(), fmt.Sprintf("margo-git-%d", time.Now().Unix()))
	}

	// Ensure directory exists, else it should be created with proper writable permissions
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	cloneDir := filepath.Join(tempDir, repoName)

	// Prepare clone options
	cloneOptions := &git.CloneOptions{
		URL:           url,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(branchOrTagName),
		SingleBranch:  true,
	}

	// Set authentication if provided
	if auth != nil {
		if auth.CABundle != nil {
			cloneOptions.CABundle = auth.CABundle
		}

		if auth.ClientCert != nil && auth.ClientKey != nil {
			cloneOptions.ClientCert = auth.ClientCert
			cloneOptions.ClientKey = auth.ClientKey
		}

		authMethod, err := getAuthMethod(url, auth)
		if err != nil {
			return "", fmt.Errorf("failed to setup authentication: %w", err)
		}
		cloneOptions.Auth = authMethod
	}

	// Clone the repository
	repo, err := git.PlainClone(cloneDir, false, cloneOptions)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository from %s: %w", url, err)
	}

	// Verify the clone was successful
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		return "", fmt.Errorf("repository clone failed: directory not found")
	}

	// Get repository info
	head, err := repo.Head()
	if err != nil {
		return cloneDir, fmt.Errorf("failed to get repository head: %w", err)
	}

	fmt.Printf("Successfully cloned repository to: %s\n", cloneDir)
	fmt.Printf("Current commit: %s\n", head.Hash())
	return cloneDir, nil
}

// getAuthMethod returns the appropriate authentication method(basic auth etc..) based on the Git URL and authentication credentials.
//
// Supported URL formats:
//   - HTTPS: https://github.com/user/repo.git
//   - HTTP: http://github.com/user/repo.git
func getAuthMethod(url string, auth *GitAuth) (transport.AuthMethod, error) {
	// SSH authentication
	if strings.HasPrefix(url, "git@") || strings.Contains(url, "ssh://") {
		return nil, fmt.Errorf("only https based git is supported")
	}

	// HTTPS authentication
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		if auth.Username != "" && auth.Token != "" {
			return &http.BasicAuth{
				Username: auth.Username,
				Password: auth.Token,
			}, nil
		}
	}

	return nil, nil
}

// extractRepoName extracts the repository name from a Git URL.
//
// Returns:
//   - string: The repository name extracted from the URL, or empty string if parsing fails
//
// Supported URL formats:
//   - HTTPS with .git suffix: https://github.com/user/repo.git
//   - HTTPS without .git suffix: https://github.com/user/repo
//   - HTTP with .git suffix: http://github.com/user/repo.git
//   - HTTP without .git suffix: http://github.com/user/repo
//   - SSH format (parsing only): git@github.com:user/repo.git
//
// Examples:
//
//	extractRepoName("https://github.com/user/myproject.git") // returns "myproject"
//	extractRepoName("https://github.com/user/myproject")     // returns "myproject"
//	extractRepoName("git@github.com:user/myproject.git")     // returns "myproject"
//	extractRepoName("invalid-url")                           // returns ""
func extractRepoName(url string) string {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// For HTTPS/HTTP URLs
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// For SSH URLs (git@host:user/repo format)
	if strings.Contains(url, ":") && !strings.Contains(url, "://") {
		parts := strings.Split(url, ":")
		if len(parts) > 1 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) > 0 {
				return pathParts[len(pathParts)-1]
			}
		}
	}

	return ""
}
