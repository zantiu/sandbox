# Make the script executable
# chmod +x generate.sh

# # Basic usage - generates client and models
# ./generate.sh --package github.com/margo/dev-repo/sdk/api

# # Generate only models
# ./generate.sh --models-only --package github.com/margo/dev-repo/sdk/api

# # Custom spec file and output directory
# ./generate.sh --spec ./api/openapi.yaml --output ./pkg/generated --package github.com/margo/dev-repo/sdk/api

# # Generate client with cleanup
# ./generate.sh --client-only --cleanup --package github.com/margo/dev-repo/sdk/api

# # Generate everything including server stub
# ./generate.sh --server-only --package github.com/margo/dev-repo/sdk/api

# # Skip validation and use custom package name
# ./generate.sh --no-validate --package github.com/margo/dev-repo/sdk/api
