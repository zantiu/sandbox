package packageManager

import (
	"context"

	"github.com/margo/dev-repo/sdk/pkg/models"
)

// TODO: we might use something like an interface to support other package sources as well
// not a priority for now.
// Source types with embedded behavior
type PackageSource[T any] interface {
	Load(ctx context.Context) (*models.AppPkg, error)
	Validate() error
	GetMetadata() T
}
