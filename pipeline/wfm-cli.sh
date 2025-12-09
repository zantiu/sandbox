#!/bin/bash
set -e

# ----------------------------
# Environment & Validation
# ----------------------------

#--- harbor settings (can be overridden via env)
EXPOSED_HARBOR_IP="${EXPOSED_HARBOR_IP:-127.0.0.1}"
EXPOSED_HARBOR_PORT="${EXPOSED_HARBOR_PORT:-8081}"

#--- symphony settings (can be overridden via env)
EXPOSED_SYMPHONY_IP="${EXPOSED_SYMPHONY_IP:-127.0.0.1}"
EXPOSED_SYMPHONY_PORT="${EXPOSED_SYMPHONY_PORT:-8082}"

#--- OCI Registry settings (can be overridden via env)
REGISTRY_URL="${REGISTRY_URL:-http://${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}}"
REGISTRY_USER="${REGISTRY_USER:-admin}"
REGISTRY_PASS="${REGISTRY_PASS:-Harbor12345}"
OCI_ORGANIZATION="${OCI_ORGANIZATION:-library}"

# ----------------------------
# Utility Functions
# ----------------------------
MAESTRO_CLI_PATH="${MAESTRO_CLI_PATH:-$HOME/symphony/cli}"
PACKAGES_DIR="${PACKAGES_DIR:-$HOME/sandbox/poc/tests/artefacts}"

# ----------------------------
# Global Package Cache
# ----------------------------
declare -a SCANNED_PACKAGE_FILES=()
declare -a SCANNED_PACKAGE_NAMES=()
declare -a SCANNED_PACKAGE_IDS=()
SCANNED_PACKAGES_COUNT=0

install_basic_utilities() {
  apt install jq -y >/dev/null 2>&1
}

check_maestro_cli() {
  if [ ! -f "${MAESTRO_CLI_PATH}/maestro" ]; then
    echo "❌ maestro CLI not found in ${MAESTRO_CLI_PATH} directory"
    echo "Please ensure maestro CLI is built and available there"
    return 1
  fi
  return 0
}

validate_choice() {
  local choice="$1"
  local max_choice="$2"
  if [[ ! "$choice" =~ ^[0-9]+$ ]] || [ "$choice" -lt 1 ] || [ "$choice" -gt "$max_choice" ]; then
    echo "❌ Invalid choice. Please enter a number between 1 and $max_choice"
    return 1
  fi
  return 0
}

# ----------------------------
# OCI Push Helper Function
# ----------------------------
push_package_to_oci() {
  local package_file="$1"
  local package_id="$2"
  local package_dir=$(dirname "$package_file")
  
  # Extract package metadata
  local package_version=$(grep -E '^\s*version:\s*' "$package_file" | head -1 | sed 's/.*version:\s*//' | tr -d '"' | tr -d "'")
  package_version="${package_version:-latest}"
  
  # Construct OCI reference
  local repository="${OCI_ORGANIZATION}/${package_id}-package"
  local tag="$package_version"
  
  echo "📦 Package Directory: $package_dir"
  echo "🏷️  OCI Reference: ${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/${repository}:${tag}"
  echo ""
  
  # Login to OCI registry
  echo "🔐 Logging into OCI registry..."
  if ! command -v oras >/dev/null 2>&1; then
    echo "❌ ORAS CLI not found. Please install ORAS first."
    return 1
  fi
  
  echo "$REGISTRY_PASS" | oras login "${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}" \
    -u "$REGISTRY_USER" --password-stdin --plain-http
  
  # Check if margo.yaml exists
  if [ ! -f "$package_file" ]; then
    echo "❌ margo.yaml not found: $package_file"
    return 1
  fi
  
  # Change to package directory to use relative paths
  cd "$package_dir" || return 1
  
  # Build file list for ORAS - start with margo.yaml
  local files=("margo.yaml:application/vnd.margo.app.description.v1+yaml")
  
  # Add resource files if directory exists and has files
  if [ -d "resources" ] && [ "$(ls -A resources 2>/dev/null)" ]; then
    while IFS= read -r file; do
      if [ -f "$file" ]; then
        files+=("$file:application/octet-stream")
      fi
    done < <(find resources -type f 2>/dev/null)
  fi
  
  # Push to OCI Registry with Margo-specific artifact type
  echo "📤 Pushing files: ${files[@]}"
  oras push "${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/${repository}:${tag}" \
    --artifact-type "application/vnd.margo.app.v1+json" \
    --plain-http \
    "${files[@]}"
  
  if [ $? -eq 0 ]; then
    echo "✅ Successfully pushed to OCI registry"
    echo "📍 Location: ${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/${repository}:${tag}"
    return 0
  else
    echo "❌ Failed to push to OCI registry"
    return 1
  fi
}


# ----------------------------
# Package Scanning Functions
# ----------------------------
scan_app_packages() {
  local packages_dir="${PACKAGES_DIR}"
  
  # Clear previous scan results
  SCANNED_PACKAGE_FILES=()
  SCANNED_PACKAGE_NAMES=()
  SCANNED_PACKAGE_IDS=()
  SCANNED_PACKAGES_COUNT=0
  
  # Define packages to exclude
  local EXCLUDED_PACKAGES=("nginx-helm" "open-telemetry-demo-helm")
  
  if [ ! -d "$packages_dir" ]; then
    echo "⚠️  Package directory not found: $packages_dir"
    echo "Creating directory..."
    mkdir -p "$packages_dir"
    return 1
  fi
  
  # Temporary array to hold all found files
  local temp_files=()
  
  # Find all margo.yaml files
  while IFS= read -r -d '' file; do
    temp_files+=("$file")
  done < <(find "$packages_dir" -maxdepth 3 -name "margo.yaml" -print0 2>/dev/null)
  
  if [ ${#temp_files[@]} -eq 0 ]; then
    return 1
  fi
  
  # Process and filter packages
  for package_file in "${temp_files[@]}"; do
    local package_dir=$(dirname "$package_file")
    
    # Get the parent directory name (the actual package name)
    local parent_dir=$(dirname "$package_dir")
    local dir_name=$(basename "$parent_dir")
    
    # If margo.yaml is directly in a directory (not in margo-package subdir)
    if [ "$(basename "$package_dir")" != "margo-package" ]; then
      dir_name=$(basename "$package_dir")
    fi
    
    # Check if package should be excluded
    local should_exclude=false
    for excluded in "${EXCLUDED_PACKAGES[@]}"; do
      if [ "$dir_name" = "$excluded" ]; then
        should_exclude=true
        break
      fi
    done
    
    # ✅ FIX: Only add to arrays if NOT excluded
    if [ "$should_exclude" = false ]; then
      SCANNED_PACKAGE_FILES+=("$package_file")
      SCANNED_PACKAGE_NAMES+=("$dir_name")
      SCANNED_PACKAGE_IDS+=("$dir_name")
    fi
  done
  
  # Update count
  SCANNED_PACKAGES_COUNT=${#SCANNED_PACKAGE_FILES[@]}
  
  if [ $SCANNED_PACKAGES_COUNT -eq 0 ]; then
    return 1
  fi
  
  return 0
}


display_scanned_packages() {
  local index=0
  
  for package_name in "${SCANNED_PACKAGE_NAMES[@]}"; do
    # Use printf to convert index to letter (a=0, b=1, etc.)
    local letter=$(printf "\\$(printf '%03o' $((97 + index)))")
    echo "   $letter) ${package_name}"
    ((index=index+1))
  done
  echo "   R) Reload the list"
}


get_scanned_package_file_path() {
  local choice="$1"
  
  # Convert letter to index
  local letter_lower=$(echo "$choice" | tr '[:upper:]' '[:lower:]')
  local array_index=$(printf '%d' "'$letter_lower")
  array_index=$((array_index - 97))  # 'a' is 97 in ASCII
  
  if [ "$array_index" -lt 0 ] || [ "$array_index" -ge "$SCANNED_PACKAGES_COUNT" ]; then
    echo ""
    return 1
  fi
  
  local package_file="${SCANNED_PACKAGE_FILES[$array_index]}"
  local package_dir=$(dirname "$package_file")
  
  # Use host:port only (strip http://)
  REGISTRY_HOST="${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}"
  
  # Substitute placeholders (FIXED: was $pkg_file, now $package_file)
  sed -i "s|{{REGISTRY_URL}}|${REGISTRY_HOST}|g" "$package_file" 2>/dev/null || true
  sed -i "s|{{REPOSITORY}}|${OCI_ORGANIZATION}/$(basename "$package_dir")-package|g" "$package_file" 2>/dev/null || true
  sed -i "s|{{TAG}}|latest|g" "$package_file" 2>/dev/null || true
  sed -i "s|{{REGISTRY_USER}}|${REGISTRY_USER}|g" "$package_file" 2>/dev/null || true
  sed -i "s|{{REGISTRY_PASS}}|${REGISTRY_PASS}|g" "$package_file" 2>/dev/null || true
  
  echo "$package_file"
  return 0
}

# ----------------------------
# Artifact Validation Helper
# ----------------------------
# ----------------------------
# Artifact Validation Helper
# ----------------------------
validate_package_artifacts() {
  local package_file="$1"
  local package_id="$2"
  
  echo "🔍 Validating artifacts for package: $package_id"
  
  # Extract deployment type from margo.yaml
  if grep -q "type: helm.v3" "$package_file"; then
    echo "📊 Detected Helm.v3 deployment type"
    
    # ============================================
    # STEP 1: Validate Helm Chart
    # ============================================
    local helm_repo=$(grep -A10 "type: helm.v3" "$package_file" | grep "repository:" | head -1 | sed 's/.*repository:\s*//' | tr -d '"' | tr -d "'")
    
    # Check for unresolved placeholders
    if [[ "$helm_repo" == *"{{HELM_REPOSITORY}}"* ]] || [[ "$helm_repo" == *"{{REPOSITORY}}"* ]]; then
      echo "❌ Helm repository placeholder not replaced in margo.yaml"
      echo "   Found: $helm_repo"
      return 1
    fi
    
    # Verify chart exists in Harbor
    echo "🔎 Checking Helm chart in Harbor: $helm_repo"
    local chart_version=$(grep -A10 "type: helm.v3" "$package_file" | grep "revision:" | head -1 | sed 's/.*revision:\s*//' | tr -d '"' | tr -d "'")
    
    if helm pull "$helm_repo" --version "${chart_version:-latest}" --plain-http 2>/dev/null; then
      rm -f *.tgz 2>/dev/null
      echo ""
      echo "✅ Helm chart already exists in Harbor OCI registry"
      echo "ℹ️  Chart: $helm_repo"
      echo "ℹ️  Version: ${chart_version:-latest}"
      echo ""
    else
      echo "❌ Helm chart not found in Harbor"
      echo ""
      echo "💡 To push your Helm chart to Harbor:"
      echo "   1. Package: helm package <chart-directory>"
      echo "   2. Push: helm push <chart>.tgz oci://${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/library --plain-http"
      return 1
    fi
    
    # ============================================
    # STEP 2: Validate Container Images for Helm
    # ============================================
    echo "🐳 Checking container images for Helm chart..."
    
    # Download and extract chart to check values.yaml
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    if helm pull "$helm_repo" --version "${chart_version:-latest}" --plain-http 2>/dev/null; then
      tar -xzf *.tgz 2>/dev/null
      local chart_dir=$(find . -maxdepth 1 -type d -name "*" | grep -v "^\.$" | head -1)
      
      if [ -f "$chart_dir/values.yaml" ]; then
        # Extract image repository and tag from values.yaml
        local image_repo=$(grep -E "^\s*repository:" "$chart_dir/values.yaml" | head -1 | sed 's/.*repository:\s*//' | tr -d '"' | tr -d "'")
        local image_tag=$(grep -E "^\s*tag:" "$chart_dir/values.yaml" | head -1 | sed 's/.*tag:\s*//' | tr -d '"' | tr -d "'")
        
        if [ -n "$image_repo" ]; then
          # Check if image is in Harbor
          if [[ "$image_repo" == "${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}"* ]]; then
            local full_image="${image_repo}:${image_tag:-latest}"
            echo "🔎 Verifying Harbor image: $full_image"
            
            # Try to inspect the image in Harbor
            if docker pull "$full_image" 2>/dev/null; then
              docker rmi "$full_image" 2>/dev/null
              echo "✅ Container image already exists in Harbor"
              echo "ℹ️  Image: $full_image"
              echo ""
            else
              echo "❌ Container image not found in Harbor: $full_image"
              echo ""
              echo "💡 To push your container image to Harbor:"
              echo "   1. Build: docker build -t $full_image <dockerfile-dir>"
              echo "   2. Login: docker login ${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT} -u admin -p Harbor12345"
              echo "   3. Push: docker push $full_image"
              cd - > /dev/null
              rm -rf "$temp_dir"
              return 1
            fi
          else
            echo "⚠️  Using public image: $image_repo:${image_tag:-latest}"
            echo "   (Not in Harbor - will be pulled from public registry)"
            echo ""
          fi
        else
          echo "⚠️  No image repository found in values.yaml"
          echo "   Chart may use default images"
          echo ""
        fi
      fi
    fi
    
    cd - > /dev/null
    rm -rf "$temp_dir"
    
  elif grep -q "type: compose" "$package_file"; then
    echo "🐳 Detected Docker Compose deployment type"
    
    # ============================================
    # STEP 1: Validate Compose File
    # ============================================
    local compose_url=$(grep -A10 "type: compose" "$package_file" | grep "packageLocation:" | head -1 | sed 's/.*packageLocation:\s*//' | tr -d '"' | tr -d "'")
    
    # Check for unresolved placeholders
    if [[ "$compose_url" == *"{{REPOSITORY}}"* ]] || [[ "$compose_url" == *"{{COMPOSE_URL}}"* ]]; then
      echo "❌ Compose URL placeholder not replaced in margo.yaml"
      echo "   Found: $compose_url"
      return 1
    fi
    
    # Verify URL is accessible
    if [[ "$compose_url" == http* ]]; then
      echo "🔎 Checking Compose file: $compose_url"
      if curl -f -s -I "$compose_url" > /dev/null 2>&1; then
        echo ""
        echo "✅ Compose file already hosted and accessible"
        echo "ℹ️  Location: $compose_url"
        echo ""
      else
        echo "❌ Compose file not accessible at: $compose_url"
        return 1
      fi
      
      # ============================================
      # STEP 2: Validate Container Images in Compose
      # ============================================
      echo "🐳 Checking container images in compose file..."
      
      # Download compose file and extract images
      local compose_content=$(curl -f -s "$compose_url" 2>/dev/null)
      
      if [ -n "$compose_content" ]; then
        # Extract all image references (simple grep - may need enhancement for complex compose files)
        local images=$(echo "$compose_content" | grep -E "^\s*image:" | sed 's/.*image:\s*//' | tr -d '"' | tr -d "'")
        
        if [ -n "$images" ]; then
          local all_images_ok=true
          
          while IFS= read -r image; do
            if [[ "$image" == "${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}"* ]]; then
              echo "🔎 Verifying Harbor image: $image"
              
              if docker pull "$image" 2>/dev/null; then
                docker rmi "$image" 2>/dev/null
                echo "✅ Container image exists in Harbor: $image"
              else
                echo "❌ Container image not found in Harbor: $image"
                echo ""
                echo "💡 To push this image to Harbor:"
                echo "   1. Pull/Build: docker pull $image OR docker build -t $image <dir>"
                echo "   2. Login: docker login ${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT} -u admin -p Harbor12345"
                echo "   3. Push: docker push $image"
                all_images_ok=false
              fi
            else
              echo "ℹ️  Using public image: $image"
              echo "   (Not in Harbor - will be pulled from public registry)"
            fi
            echo ""
          done <<< "$images"
          
          if [ "$all_images_ok" = false ]; then
            return 1
          fi
        else
          echo "⚠️  No images found in compose file"
          echo ""
        fi
      fi
    fi
  else
    echo "⚠️  Unknown deployment type in margo.yaml"
    return 1
  fi
  
  echo "✅ All artifacts validated successfully"
  echo ""
  echo "📋 Summary:"
  echo "   ✓ Helm charts/Compose files: Already in Harbor/accessible"
  echo "   ✓ Container images: Verified in Harbor OCI registry"
  echo "   ✓ Package metadata: Ready for upload"
  echo ""
  
  return 0
}


# ----------------------------
# App Supplier Functions
# ----------------------------

supplier_upload_package() {
  clear
  echo "📤 Upload App Package"
  echo "===================="
  echo ""
  
  # Define default apps that have templates in symphony/cli/templates
  local DEFAULT_APPS=("custom-otel-helm-app" "nextcloud-compose")
  
  while true; do
    echo -e "\033[1;33m[Place your already prepared Margo Application Package in $PACKAGES_DIR to auto-list here.]\033[0m:"
    echo ""
    echo -e "\033[1;33mOR\033[0m"
    echo ""
    echo -e "\033[1;33m[To create a new package, go back to previous menu and use '4) Help in Creating a Package Locally' option.]\033[0m"
    echo ""
    
    if ! scan_app_packages; then
      echo "⚠️  No packages found in $PACKAGES_DIR"
      echo ""
      read -p "Press Enter to go back..." 
      return
    fi
    echo -e "Scanned Application Packages:\n"
    display_scanned_packages
    echo ""
    echo "   Q) Go Back"
    echo ""
    echo -e "\033[1;33m[Note: a and b are pre-existing Application Packages in the Sandbox.]\033[0m"
    echo ""
    read -p "Choose to upload [a-z, R, Q]: " choice
    
    choice_lower=$(echo "$choice" | tr '[:upper:]' '[:lower:]')
    
    if [ "$choice_lower" = "q" ]; then
      return
    fi
    
    if [ "$choice_lower" = "r" ]; then
      clear
      echo "📤 Upload App Package"
      echo "===================="
      echo ""
      echo "🔄 Reloading package list..."
      continue
    fi
    
    package_file=$(get_scanned_package_file_path "$choice_lower")
    
    if [ -z "$package_file" ] || [ ! -f "$package_file" ]; then
      echo "❌ Invalid choice"
      sleep 1
      continue
    fi
    
    # Get package details
    local array_index=$(printf '%d' "'$choice_lower")
    array_index=$((array_index - 97))
    local package_name="${SCANNED_PACKAGE_NAMES[$array_index]}"
    local package_id="${SCANNED_PACKAGE_IDS[$array_index]}"
    local package_dir=$(dirname "$package_file")
    local parent_dir=$(dirname "$package_dir")
    
    # Check if this is a default app
    local is_default_app=false
    for default_app in "${DEFAULT_APPS[@]}"; do
      if [ "$package_id" = "$default_app" ]; then
        is_default_app=true
        break
      fi
    done
    
    echo ""
    
    # ============================================================
    # STEP 2: Handle OCI push based on app type with validation
    # ============================================================
    if [ "$is_default_app" = false ]; then
      # Custom app - validate artifacts FIRST
      echo "🔍 Validating package artifacts before upload..."
      echo ""
      
      if ! validate_package_artifacts "$package_file" "$package_id"; then
        echo ""
        echo "❌ Package validation failed"
        echo ""
        echo "📋 Pre-upload checklist:"
        echo "   ✓ Helm charts must be pushed to Harbor OCI registry"
        echo "   ✓ Docker Compose files must be accessible via URL"
        echo "   ✓ Container images must be pushed to Harbor"
        echo "   ✓ All {{PLACEHOLDERS}} in margo.yaml must be replaced"
        echo ""
        echo "💡 Use option '4) Help in Creating a Package Locally' for guidance"
        echo ""
        read -p "Press Enter to go back..."
        continue
      fi
      
      # Validation passed - now push margo.yaml + resources to OCI
      echo ""
      echo "✅ Pushing Margo Application Package metadata to OCI repository..."
      if ! push_package_to_oci "$package_file" "$package_id"; then
        echo "❌ OCI push failed"
        read -p "Press Enter to go back..."
        continue
      fi
      echo ""
    else
      # Default app - skip validation and OCI push
      echo "ℹ️  Pre-existing Application Package: $package_id"
      echo "⏭️  Skipping validation and OCI push (already managed by wfm.sh)"
      echo ""
    fi
    
    # ============================================================
    # Determine which package.yaml to use for WFM upload
    # ============================================================
    local wfm_package_file=""
    
    if [ "$is_default_app" = true ]; then
      # Use existing template from symphony/cli/templates
      case $package_id in
        "custom-otel-helm-app")
          wfm_package_file="$HOME/symphony/cli/templates/margo/custom-otel-helm/package.yaml.copy"
          cp "$HOME/symphony/cli/templates/margo/custom-otel-helm/package.yaml" "$wfm_package_file"
          ;;
        "nextcloud-compose")
          wfm_package_file="$HOME/symphony/cli/templates/margo/nextcloud-compose/package.yaml.copy"
          cp "$HOME/symphony/cli/templates/margo/nextcloud-compose/package.yaml" "$wfm_package_file"
          ;;
        *)
          echo "⚠️  No template found for default Application Package: $package_id"
          read -p "Press Enter to go back..."
          continue
          ;;
      esac
    else
      # ✅ Custom app - use shared template from symphony directory
      local template_file="$HOME/symphony/cli/templates/margo/package.yaml.template"
      
      if [ ! -f "$template_file" ]; then
        echo "❌ Shared package.yaml template not found: $template_file"
        echo ""
        echo "💡 Creating template automatically..."
        
        # Create the template directory if it doesn't exist
        mkdir -p "$HOME/symphony/cli/templates/margo"
        
        # Create the template
        cat > "$template_file" << 'TEMPLATE_EOF'
# This is an input template allowing the WFM get OCI-based application packages and store in its DB required for further deployment.
# This file is not MARGO specified.
apiVersion: non-margo.org
kind: ApplicationPackage
metadata:
  name: {{PACKAGE_ID}}
  labels:
    env: prod
  annotations:
    description: "{{DESCRIPTION}}"
spec:
  sourceType: OCI_REPO
  source:
    registryUrl: "{{REGISTRY_URL}}"
    repository: "{{REPOSITORY}}"
    tag: "{{TAG}}"
    authentication:
      type: "basic"
      username: "{{REGISTRY_USER}}"
      password: "{{REGISTRY_PASS}}"
TEMPLATE_EOF
        
        if [ ! -f "$template_file" ]; then
          echo "❌ Failed to create template"
          read -p "Press Enter to go back..."
          continue
        fi
        
        echo "✅ Template created successfully"
        echo ""
      fi
      
      # Create working copy with package-specific values
      wfm_package_file="/tmp/package-${package_id}.yaml"
      cp "$template_file" "$wfm_package_file"
      
      # Replace package-specific placeholders
      sed -i "s|{{PACKAGE_ID}}|${package_id}|g" "$wfm_package_file"
      
      # Extract description from margo.yaml
      local description=$(grep -A5 "metadata:" "$package_file" | grep "description:" | head -1 | sed 's/.*description:\s*//' | tr -d '"' | tr -d "'")
      sed -i "s|{{DESCRIPTION}}|${description:-Custom application package}|g" "$wfm_package_file"
    fi
    
    # Verify package.yaml exists
    if [ -z "$wfm_package_file" ] || [ ! -f "$wfm_package_file" ]; then
      echo "⚠️  package.yaml not found"
      echo "ℹ️  Expected location: $wfm_package_file"
      echo "💡 Use the Application Package Creation Wizard to generate it"
      echo ""
      read -p "Press Enter to go back..."
      continue
    fi
    
    # Upload to WFM marketplace
    echo "✅ Uploading to WFM marketplace..."

    # Substitute placeholders (matching template structure)
    REGISTRY_HOST="${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}"
    sed -i "s|{{REGISTRY_URL}}|${REGISTRY_HOST}|g" "$wfm_package_file"
    sed -i "s|{{REPOSITORY}}|${OCI_ORGANIZATION}/${package_id}-package|g" "$wfm_package_file"

    # ✅ Conditional tag replacement based on app type
    if [ "$is_default_app" = false ]; then
      # Custom apps: Extract version from margo.yaml
      local package_version=$(grep -E '^\s*version:\s*' "$package_file" | head -1 | sed 's/.*version:\s*//' | tr -d '"' | tr -d "'")
      package_version="${package_version:-latest}"
      sed -i "s|{{TAG}}|${package_version}|g" "$wfm_package_file"
    else
      # Pre-existing apps: Use "latest" (matches what wfm.sh pushes)
      sed -i "s|{{TAG}}|latest|g" "$wfm_package_file"
    fi

    sed -i "s|{{REGISTRY_USER}}|${REGISTRY_USER}|g" "$wfm_package_file"
    sed -i "s|{{REGISTRY_PASS}}|${REGISTRY_PASS}|g" "$wfm_package_file"

    
    if check_maestro_cli; then
      # Capture both stdout and stderr
      upload_output=$(${MAESTRO_CLI_PATH}/maestro wfm apply -f "$wfm_package_file" 2>&1)
      upload_exit_code=$?
      
      echo "$upload_output"
      
      # Check for actual success (no "failed" or "error" in output)
      if [ $upload_exit_code -eq 0 ] && ! echo "$upload_output" | grep -qi "failed\|error"; then
        echo ""
        echo "   Package ID: $package_id"
        echo "✅ Package uploaded to WFM marketplace successfully!"
      else
        echo ""
        echo "❌ Failed to upload package to WFM marketplace"
        if echo "$upload_output" | grep -q "unsupported kind"; then
          echo "💡 Hint: Ensure package.yaml has kind: ApplicationPackage"
        fi
      fi
      
      # Clean up copy for default apps and temp files for custom apps
      if [ "$is_default_app" = true ]; then
        rm -f "$wfm_package_file"
      else
        # Clean up temp file for custom apps
        rm -f "/tmp/package-${package_id}.yaml"
      fi
    fi
    
    echo ""
    read -p "Press Enter to go back..."
    return
  done
}


supplier_list_packages() {
  clear
  echo "📦 List App Packages"
  echo "===================="
  echo ""
  
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list app-pkg || echo "❌ Failed to list packages"
  fi
  
  echo ""
  read -p "Press Enter to go back..."
}

# Helper function to get package name from UUID
get_package_name_from_id() {
  local pkg_id="$1"
  
  if ! check_maestro_cli; then
    return 1
  fi
  
  local app_packages=$(${MAESTRO_CLI_PATH}/maestro wfm list app-pkg -o json 2>/dev/null)
  
  if [ $? -ne 0 ] || [ -z "$app_packages" ]; then
    return 1
  fi
  
  if command -v jq >/dev/null 2>&1; then
    local pkg_name=$(echo "$app_packages" | jq -r --arg id "$pkg_id" '
      .Data[0].items[] | 
      select(.metadata.id == $id) | 
      .metadata.name
    ')
    
    if [ -n "$pkg_name" ] && [ "$pkg_name" != "null" ]; then
      echo "$pkg_name"
      return 0
    fi
  fi
  
  return 1
}


# ----------------------------
# supplier_delete_package function
# ----------------------------
supplier_delete_package() {
  clear
  echo "🗑️  Delete App Package"
  echo "====================="
  echo ""
  
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list app-pkg
  fi
  
  echo ""
  echo "💡 Enter the package ID (UUID from first column, not the name)"
  read -p "Package ID to delete: " package_id
    
  if [ -z "$package_id" ]; then
    echo "❌ Package ID is required"
    sleep 2
    return
  fi
  
  echo ""
  read -p "Are you sure you want to delete '$package_id'? (y/n): " confirm
  
  if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Deletion cancelled"
    sleep 1
    return
  fi
  
  echo ""
  echo "🗑️  Deleting package details from WFM marketplace..."
  if check_maestro_cli; then
    if ${MAESTRO_CLI_PATH}/maestro wfm delete app-pkg "$package_id" 2>&1 | grep -q "successfully\|deleted"; then
      echo "✅ Done"
    else
      echo "❌ Failed to delete from WFM marketplace"
    fi
  fi
  
  echo "🗑️  Deleting package from OCI repository..."
  # Only delete from OCI for non pre-existing ones
  local DEFAULT_APPS=("custom-otel-helm-app" "nextcloud-compose")
  local is_default_app=false

  for default_app in "${DEFAULT_APPS[@]}"; do
    if [ "$package_id" = "$default_app" ]; then
      is_default_app=true
      break
    fi
  done

  if [ "$is_default_app" = false ]; then
    echo ""
    echo "🗑️  Deleting custom package from OCI repository..."
    
    # ✅ Resolve UUID to package name for OCI lookup
    echo "🔍 Resolving package name from UUID..."
    local package_name=$(get_package_name_from_id "$package_id")
    
    if [ -z "$package_name" ]; then
      echo "⚠️  Could not resolve package name from UUID: $package_id"
      echo "   Skipping OCI deletion (may require manual cleanup via Harbor UI)"
      echo ""
      read -p "Press Enter to go back..."
      return
    fi
    
    echo "✅ Resolved to package: $package_name"
    
    # Construct OCI repository path using package name
    local oci_repo="${OCI_ORGANIZATION}/${package_name}"
    if [[ ! "$package_name" =~ -package$ ]]; then
      oci_repo="${oci_repo}-package"
    fi
    
    echo "📍 OCI location: ${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/${oci_repo}"
    echo ""
    
    read -p "Delete OCI artifacts? (y/n): " delete_oci
    
    if [[ "$delete_oci" =~ ^[Yy]$ ]]; then
      # Login to OCI registry
      echo "$REGISTRY_PASS" | oras login "${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}" \
        -u "$REGISTRY_USER" --password-stdin --plain-http 2>/dev/null
      
      # Get all tags for this repository
      echo "📋 Finding package versions..."
      local tags=$(oras repo tags "${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/${oci_repo}" --plain-http 2>/dev/null)
      
      if [ -z "$tags" ]; then
        echo "⚠️  No OCI artifacts found for: $oci_repo"
      else
        echo "Found versions: $tags"
        echo ""
        
        # Delete each tag using ORAS
        while IFS= read -r tag; do
          if [ -n "$tag" ]; then
            echo "🗑️  Deleting ${oci_repo}:${tag}..."
            
            # Use ORAS to delete the manifest
            if oras manifest delete "${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/${oci_repo}:${tag}" --plain-http 2>/dev/null; then
              echo "✅ Deleted version: $tag"
            else
              echo "⚠️  Could not delete version: $tag (may require manual cleanup)"
            fi
          fi
        done <<< "$tags"
        
        echo ""
        echo "✅ OCI package deletion completed"
      fi
    else
      echo "ℹ️  OCI artifacts preserved in Harbor"
    fi
  else
    echo ""
    echo "ℹ️  Pre-existing Application Package - OCI artifacts managed by wfm.sh"
    echo "   Skipping OCI deletion"
  fi

  echo ""
  read -p "Press Enter to go back..."
}


supplier_package_creation_wizard() {
  clear
  echo "📦 App Package Creation Wizard"
  echo "=============================="
  echo ""
  echo "This wizard helps you create a Margo Application Package step-by-step."
  echo ""
  echo "Select Deployment Profile:"
  echo ""
  echo "1) Helm.v3"
  echo "   → Use this if your application is packaged as a Helm chart"
  echo "   → ⚠️  Currently supports ONLY local Harbor registry"
  echo "   → Charts must be pushed to: oci://172.19.59.148:8081/library/<chart-name>"
  echo "   → Example: Kubernetes applications, microservices"
  echo ""
  echo "2) Docker Compose"
  echo "   → Use this if your application uses docker-compose.yaml"
  echo "   → ⚠️  Compose file must be publicly accessible via HTTP/HTTPS"
  echo "   → Example URL: https://raw.githubusercontent.com/docker/awesome-compose/refs/heads/master/nextcloud-redis-mariadb/compose.yaml"
  echo "   → Example: Multi-container applications, development environments"
  echo ""
  
  read -p "Enter choice [1-2]: " profile_choice
  
  local deployment_type=""
  local deployment_url=""
  
  case $profile_choice in
    1)
      deployment_type="helm.v3"
      echo ""
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo "📦 Helm Chart Configuration (Local Harbor Only)"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo ""
      echo "⚠️  IMPORTANT: Only local Harbor registry is currently supported!"
      echo ""
      echo "Your Helm chart OCI URL MUST use this format:"
      echo "   oci://172.19.59.148:8081/library/<your-chart-name>"
																			   
																 
      echo ""
      echo "💡 Example:"
      echo "   oci://172.19.59.148:8081/library/nginx-sample"
      echo "   oci://172.19.59.148:8081/library/my-app"
      echo ""
      echo "📋 Before proceeding, ensure you have:"
      echo "   1. Created your Helm chart: helm create <chart-name>"
      echo "   2. Packaged it: helm package <chart-name>"
      echo "   3. Pushed to Harbor: helm push <chart>.tgz oci://172.19.59.148:8081/library --plain-http"
      echo ""
      read -p "Enter OCI URL of your Helm chart: " deployment_url
      
      if [ -z "$deployment_url" ]; then
        echo "❌ Helm chart URL is required"
        sleep 2
        return
      fi
      
      # Validate it's using local Harbor
      if [[ ! "$deployment_url" =~ ^oci://172\.19\.59\.148:8081/library/ ]]; then
        echo ""
        echo "❌ Invalid URL format!"
        echo "   Expected: oci://172.19.59.148:8081/library/<chart-name>"
        echo "   Got: $deployment_url"
        echo ""
        read -p "Continue anyway? (y/n): " continue_anyway
        if [[ ! "$continue_anyway" =~ ^[Yy]$ ]]; then
          return
        fi
      fi
      ;;
      
    2)
      deployment_type="compose"
      echo ""
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo "🐳 Docker Compose Configuration (Public URL Required)"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo ""
      echo "⚠️  IMPORTANT: Compose file must be publicly accessible!"
      echo ""
      echo "Your compose file URL must:"
      echo "   • Be accessible via HTTP/HTTPS without authentication"
      echo "   • Return raw YAML content (not HTML)"
      echo "   • Be reachable from the deployment device"
      echo ""
      echo "💡 Recommended hosting options:"
      echo "   • GitHub raw: https://raw.githubusercontent.com/user/repo/main/compose.yaml"
      echo "   • GitLab raw: https://gitlab.com/user/repo/-/raw/main/compose.yaml"
      echo "   • Public web server: http://yourserver.com/apps/compose.yaml"
      echo ""
      echo "📋 Quick test your URL:"
      echo "   curl -f <your-url> | head -5"
      echo "   (Should show YAML content, not HTML)"
      echo ""
      read -p "Enter URL of your compose file: " deployment_url
      
      if [ -z "$deployment_url" ]; then
        echo "❌ Compose file URL is required"
        sleep 2
        return
      fi
      
      # Validate URL format
      if [[ ! "$deployment_url" =~ ^https?:// ]]; then
        echo "❌ URL must start with 'http://' or 'https://'"
        sleep 2
												
        return
      fi
		
      
      # Test if URL is accessible
      echo ""
      echo "🔍 Testing URL accessibility..."
      if curl -f -s -I "$deployment_url" > /dev/null 2>&1; then
        echo "✅ URL is accessible"
        
        # Check if it returns YAML (not HTML)
        local content_type=$(curl -f -s -I "$deployment_url" 2>/dev/null | grep -i "content-type" | head -1)
        if [[ "$content_type" == *"text/html"* ]]; then
          echo "⚠️  Warning: URL returns HTML, not YAML"
          echo "   Make sure you're using the 'raw' file URL"
          read -p "Continue anyway? (y/n): " continue_anyway
          if [[ ! "$continue_anyway" =~ ^[Yy]$ ]]; then
            return
          fi
        fi
      else
        echo "❌ URL is not accessible!"
        echo ""
        echo "💡 Common issues:"
        echo "   • URL requires authentication (not supported)"
        echo "   • URL is not publicly accessible"
        echo "   • Firewall blocking access"
        echo ""
        read -p "Continue anyway? (y/n): " continue_anyway
        if [[ ! "$continue_anyway" =~ ^[Yy]$ ]]; then
          return
        fi
      fi
      ;;
      
    *)
      echo "❌ Invalid choice"
      sleep 2
      return
      ;;
  esac
  
   # Rest of the wizard continues with metadata collection...
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "📝 Application Metadata"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  
  # Collect package metadata
  echo "💡 Application ID should be unique (e.g., com-vendor-app-name)"
  read -p "Enter application ID: " app_id
  
  if [ -z "$app_id" ]; then
    echo "❌ Application ID is required"
    sleep 2
    return
  fi
  
  echo ""
  echo "💡 This is the display name users will see"
  read -p "Enter application name: " app_name
  
  if [ -z "$app_name" ]; then
    echo "❌ Application name is required"
    sleep 2
    return
  fi
  
  echo ""
  echo "💡 Brief description of what your application does"
  read -p "Enter description: " app_description
  
  if [ -z "$app_description" ]; then
    echo "❌ Description is required"
    sleep 2
    return
  fi
  
  echo ""
  echo "💡 Semantic version (e.g., 1.0.0, 2.1.3)"
  read -p "Enter version (default: 1.0.0): " app_version
  app_version="${app_version:-1.0.0}"
  
  echo ""
  echo "💡 Short one-liner about your app (optional)"
  read -p "Enter tagline (press Enter to use description): " app_tagline
  app_tagline="${app_tagline:-$app_description}"
  
  echo ""
  echo "💡 Website or documentation URL (optional)"
  read -p "Enter website URL (default: https://example.com): " app_site
  app_site="${app_site:-https://example.com}"
  
  echo ""
  echo "💡 Author/maintainer information"
  read -p "Enter author name (default: Development Team): " author_name
  author_name="${author_name:-Development Team}"
  
  echo ""
  read -p "Enter author email (default: dev@example.com): " author_email
  author_email="${author_email:-dev@example.com}"
  
  echo ""
  read -p "Enter organization name (default: Example Organization): " org_name
  org_name="${org_name:-Example Organization}"
  
  # Create package directory
  local package_id=$(echo "$app_id" | tr '[:upper:]' '[:lower:]')
  local package_dir="${PACKAGES_DIR}/${app_name}"
  
  if [ -d "$package_dir" ]; then
    echo ""
    echo "⚠️  Package directory already exists: $package_dir"
    read -p "Overwrite? (y/n): " overwrite
    if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
      echo "Wizard cancelled"
      sleep 1
      return
    fi
    rm -rf "$package_dir"
  fi
  
  mkdir -p "$package_dir/resources"
  
  # Create margo.yaml template
  local template_file="${package_dir}/margo.yaml"
  
  cat > "$template_file" << EOF
apiVersion: margo.org/v1-alpha1
kind: ApplicationDescription
metadata:
  id: ${app_id}
  name: ${app_name}
  description: ${app_description}
  version: ${app_version}
  catalog:
    application:
      icon: ./resources/app-logo.png # no need to change this
      tagline: ${app_tagline}
      descriptionFile: ./resources/description.md # no need to change this
      releaseNotes: ./resources/release-notes.md # no need to change this
      licenseFile: ./resources/license.txt # no need to change this
      site: ${app_site}
      tags: ["application"]
    author:
      - name: ${author_name}
        email: ${author_email}
    organization:
      - name: ${org_name}
        site: ${app_site}
deploymentProfiles:
EOF

  # Add deployment profile based on type
  if [ "$deployment_type" = "helm.v3" ]; then
    local component_name=$(echo "$app_name" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
    cat >> "$template_file" << EOF
  - type: helm.v3
    id: ${app_id}-helm-v3
    description: Helm chart deployment for ${app_name}
    components:
      - name: ${component_name}
        properties:
          repository: ${deployment_url}
          revision: ${app_version}
          wait: true
          timeout: 5m
    requiredResources: # this section doesn't have any impact as of now
      cpu:
        cores: 1.0
        architectures:
          - amd64
      memory: 512Mi
      storage: 1Gi
EOF
  elif [ "$deployment_type" = "compose" ]; then
    local component_name=$(echo "$app_name" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
    cat >> "$template_file" << EOF
  - type: compose
    id: ${app_id}-compose
    components:
      - name: ${component_name}-stack
        properties:
          packageLocation: ${deployment_url}
EOF
  fi

  # Add basic parameters section
  cat >> "$template_file" << EOF
parameters: # this section is examplary, please replace with the parameters as per your application
  appPort:
    value: 8080
    targets:
      - pointer: PORTS.8080
        components: ["${component_name}-stack"]
EOF

  # Add configuration section
  cat >> "$template_file" << EOF
configuration: # this section is examplary, please replace with the configurations section as per your application
  sections:
    - name: General Settings
      settings:
        - parameter: appPort
          name: Application Port
          description: Port to expose the application
          schema: portRange
  schema:
    - name: requiredText
      dataType: string
      maxLength: 45
      allowEmpty: false
    - name: portRange
      dataType: integer
      minValue: 1024
      maxValue: 65535
      allowEmpty: false
    - name: cpuRange
      dataType: double
      minValue: 0.5
      maxPrecision: 1
      allowEmpty: false
    - name: memoryRange
      dataType: integer
      minValue: 512
      allowEmpty: false
EOF

  # Create resource files
  cat > "${package_dir}/resources/description.md" << EOF
# ${app_name}

${app_description}

## Features

- Feature 1
- Feature 2
- Feature 3

## Requirements

- Minimum CPU: 1 core
- Minimum Memory: 512Mi
- Minimum Storage: 1Gi
EOF

  cat > "${package_dir}/resources/release-notes.md" << EOF
# Release Notes - ${app_version}

## What's New

- Initial release

## Bug Fixes

- None

## Known Issues

- None
EOF

  cat > "${package_dir}/resources/license.txt" << EOF
Copyright (c) $(date +%Y) ${org_name}

All rights reserved.
EOF

  # Create placeholder icon
  touch "${package_dir}/resources/app-logo.png"
  
  # Open template in editor for customization
  echo ""
  echo "📝 Please edit the following app description template (opening in system default editor)..."
  sleep 2
  
  EDITOR="${EDITOR:-${VISUAL:-vi}}"
  $EDITOR "$template_file"
  
 
  
  # Basic YAML validation
  if command -v grep >/dev/null 2>&1; then
    if ! grep -q "apiVersion: margo.org/v1-alpha1" "$template_file" || ! grep -q "kind: ApplicationDescription" "$template_file"; then
      echo "❌ Invalid margo.yaml format"
      sleep 2
      return
    fi
  fi

  # Final confirmation
  echo ""
  echo "The package will be created in the directory: $package_dir"
  read -p "Should proceed? (y/n): " proceed
  
  if [[ ! "$proceed" =~ ^[Yy]$ ]]; then
    echo "Wizard cancelled"
    rm -rf "$package_dir"
    sleep 1
    return
  fi
  
    echo ""
    echo "✅ Package created successfully!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📁 Package Location:"
    echo "   $package_dir"
    echo ""
    echo "📄 Files Created:"
    echo "   ✓ margo.yaml (Application description)"
    echo "   ✓ resources/ (Icon, docs, license)"
    echo ""
    echo "📝 Note: Resource files are placeholders - customize as needed"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 Next Steps:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1️⃣  Go back to the App Supplier Menu"
    echo ""
    echo "2️⃣  Select: 1) Upload App Package"
    echo ""
    echo "3️⃣  Verify your package appears in the scanned list:"
    echo "   → Look for: ${app_name}"
    echo "   → It should be listed with a letter (e.g., 'c) ${app_name}')"
    echo ""
    echo "4️⃣  Select your package letter to upload"
    echo ""
    echo "5️⃣  The system will:"
    echo "   • Validate all artifacts (Helm charts, container images)"
    echo "   • Push margo.yaml + resources to OCI registry"
    echo "   • Register package with WFM marketplace"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
  
  read -p "Press Enter to go back to App Supplier Menu..."


  
  read -p "Press Enter to go back..."
}

show_supplier_menu() {
  while true; do
    clear
    echo "📦 App Supplier Menu"
    echo "===================="
    echo ""
    echo "1) Upload App Package"
    echo "2) List App Packages"
    echo "3) Delete App Package"
    echo "4) Help in Creating a Package Locally"
    echo "5) Go Back"
    echo ""
    
    read -p "Enter choice [1-5]: " choice
    
    case $choice in
      1) supplier_upload_package ;;
      2) supplier_list_packages ;;
      3) supplier_delete_package ;;
      4) supplier_package_creation_wizard ;;
      5) return ;;
      *) echo "⚠️ Invalid choice"; sleep 1 ;;
    esac
  done
}

# ----------------------------
# End-User Functions
# ----------------------------
enduser_list_packages() {
  clear
  echo "📦 List App Packages"
  echo "===================="
  echo ""
  echo "Following apps are available on WFM marketplace:"
  echo ""
  
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list app-pkg || echo "❌ Failed to list packages"
  fi
  
  echo ""
  read -p "Press Enter to go back..."
}

enduser_list_devices() {
  clear
  echo "🖥️  List Devices"
  echo "==============="
  echo ""
  
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list devices || echo "❌ Failed to list devices"
  fi
  
  echo ""
  read -p "Press Enter to go back..."
}

enduser_list_deployments() {
  clear
  echo "🚀 List Deployments"
  echo "==================="
  echo ""
  
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list deployment || echo "❌ Failed to list deployments"
  fi
  
  echo ""
  read -p "Press Enter to go back..."
}

enduser_list_all() {
  clear
  echo "📋 List All Resources"
  echo "====================="
  echo ""
  
  echo "📦 App Packages:"
  echo "----------------"
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list app-pkg || echo "❌ Failed to list packages"
  fi
  
  echo ""
  echo "🖥️  Devices:"
  echo "----------"
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list devices || echo "❌ Failed to list devices"
  fi
  
  echo ""
  echo "🚀 Deployments:"
  echo "---------------"
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list deployment || echo "❌ Failed to list deployments"
  fi
  
  echo ""
  read -p "Press Enter to go back..."
}

# ----------------------------
# Dynamic Instance File Generation
# ----------------------------
generate_instance_file_from_margo() {
  local package_id="$1"
  local package_name="$2"
  
  echo "📝 Generating instance.yaml for custom package: $package_name"
  
  # Find margo.yaml for this package
  local margo_file=$(find "$PACKAGES_DIR" -path "*/${package_name}/margo-package/margo.yaml" -o -path "*/${package_name}/margo.yaml" 2>/dev/null | head -1)
  
  if [ -z "$margo_file" ] || [ ! -f "$margo_file" ]; then
    echo "❌ margo.yaml not found for package: $package_name" >&2
    return 1
  fi
  
  # Create temp instance file
  local instance_file="/tmp/instance-${package_id}.yaml"
  
  # Extract deployment profile details from margo.yaml
  local deployment_type=$(grep -A20 "deploymentProfiles:" "$margo_file" | grep "type:" | head -1 | sed 's/.*type:\s*//' | tr -d '"' | tr -d "'")
  local component_name=$(grep -A20 "deploymentProfiles:" "$margo_file" | grep "name:" | head -1 | sed 's/.*name:\s*//' | tr -d '"' | tr -d "'")
  local revision=$(grep -A20 "deploymentProfiles:" "$margo_file" | grep "revision:" | head -1 | sed 's/.*revision:\s*//' | tr -d '"' | tr -d "'")
  
  # Generate instance.yaml based on deployment type
  if [ "$deployment_type" = "helm.v3" ]; then
    cat > "$instance_file" << EOF
apiVersion: margo.org/v1-alpha1
kind: ApplicationDeployment
metadata:
  name: ${package_name}-instance
spec:
  appPackageRef:
    id: {{PACKAGE_ID}}
  deviceRef:
    id: {{DEVICE_ID}}
  deploymentProfile:
    type: helm.v3
    components:
      - name: ${component_name}
        properties:  
          repository: {{REPOSITORY}}
          revision: ${revision:-latest}
          wait: true
          timeout: 5m
  parameters: {}
EOF
  elif [ "$deployment_type" = "compose" ]; then
    cat > "$instance_file" << EOF
apiVersion: margo.org/v1-alpha1
kind: ApplicationDeployment
metadata:
  name: ${package_name}-instance
spec:
  appPackageRef:
    id: {{PACKAGE_ID}}
  deviceRef:
    id: {{DEVICE_ID}}
  deploymentProfile:
    type: compose
    components:
      - name: ${component_name}
        properties:  
          packageLocation: {{REPOSITORY}}
  parameters: {}
EOF
  else
    echo "❌ Unknown deployment type: $deployment_type" >&2
    return 1
  fi
  
  echo "✅ Generated instance file: $instance_file"
  echo "$instance_file"
  return 0
}


get_instance_file_path() {
  local package_name="$1"
  local package_id="$2"  # ✅ Now accepts package_id parameter
  local file_path=""
  
  if [ -z "$HOME" ]; then
    echo "❌ HOME environment variable not set" >&2
    return 1
  fi
  
  # Check for pre-existing templates first
  case $package_name in
    "custom-otel-helm-app"|"custom-otel"|"otel-demo-pkg")
      original_file_path="$HOME/symphony/cli/templates/margo/custom-otel-helm/instance.yaml"
      file_path="$HOME/symphony/cli/templates/margo/custom-otel-helm/instance.yaml.copy"
      cp -f ${original_file_path} ${file_path} 2>/dev/null || true ;;
    
    "nextcloud-compose-app"|"nextcloud"|"nextcloud-pkg")
      original_file_path="$HOME/symphony/cli/templates/margo/nextcloud-compose/instance.yaml"
      file_path="$HOME/symphony/cli/templates/margo/nextcloud-compose/instance.yaml.copy"
      cp -f ${original_file_path} ${file_path} 2>/dev/null || true ;;
    
    *)
      # ✅ NEW: For custom packages, generate instance.yaml dynamically
      echo "📝 Custom package detected: $package_name"
      
      file_path=$(generate_instance_file_from_margo "$package_id" "$package_name")
      
      if [ $? -ne 0 ] || [ -z "$file_path" ]; then
        echo "❌ Failed to generate instance file" >&2
        return 1
      fi
      ;;
  esac
  
  if [ -f "$file_path" ]; then
    echo "$file_path"
    return 0
  else
    echo "❌ Deployment file not found: $file_path" >&2
    return 1
  fi
}


get_oci_repository_path() {
  local package_name="$1"
  local package_id="$2"  # ✅ Now accepts package_id parameter
  local container_url=""
  
  # Check for pre-existing mappings first
  case $package_name in
    "custom-otel-helm-app"|"custom-otel"|"otel-demo-pkg")
      container_url="oci://${EXPOSED_HARBOR_IP}:${EXPOSED_HARBOR_PORT}/library/custom-otel-helm" ;;
    
    "nextcloud-compose-app"|"nextcloud"|"nextcloud-pkg")
      container_url="https://raw.githubusercontent.com/docker/awesome-compose/refs/heads/master/nextcloud-redis-mariadb/compose.yaml" ;;
    
    *)
        # ✅ For custom packages, extract from margo.yaml
    local margo_file=$(find "$PACKAGES_DIR" -path "*/${package_name}/margo-package/margo.yaml" -o -path "*/${package_name}/margo.yaml" 2>/dev/null | head -1)
    
    if [ -f "$margo_file" ]; then
      # Extract repository value more carefully to preserve full OCI URL
      container_url=$(grep -A20 "deploymentProfiles:" "$margo_file" | grep -E "^\s*repository:" | head -1 | awk '{print $2}' | tr -d '"' | tr -d "'")
      
      # If not found, try packageLocation for compose
      if [ -z "$container_url" ]; then
        container_url=$(grep -A20 "deploymentProfiles:" "$margo_file" | grep -E "^\s*packageLocation:" | head -1 | awk '{print $2}' | tr -d '"' | tr -d "'")
      fi
    fi
    ;;
  esac
  
  echo "$container_url"
}


enduser_deploy_instance() {
  clear
  echo "🚀 Deploy Instance"
  echo "=================="
  echo ""
  
  # List available packages
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list app-pkg
  fi
  
  echo ""
  read -p "Enter the package ID: " package_id
  
  if [ -z "$package_id" ]; then
    echo "❌ Package ID is required"
    sleep 2
    return
  fi
  
  echo ""
  # List available devices
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list devices
  fi
  
  echo ""
  read -p "Enter the device ID: " device_id
  
  if [ -z "$device_id" ]; then
    echo "❌ Device ID is required"
    sleep 2
    return
  fi
  
  # Get package details
  echo ""
  echo "📋 Getting package details..."
  app_packages=$(${MAESTRO_CLI_PATH}/maestro wfm list app-pkg -o json 2>/dev/null)
  
  if [ $? -ne 0 ] || [ -z "$app_packages" ]; then
    echo "❌ Failed to get package list"
    sleep 2
    return
  fi
  
  # Parse JSON to find the package
  if command -v jq >/dev/null 2>&1; then
    package_name=$(echo "$app_packages" | jq -r --arg pkg_id "$package_id" '
      .Data[0].items[] | 
      select(.metadata.id == $pkg_id or .metadata.name == $pkg_id) | 
      .metadata.name
    ')
    
    if [ -z "$package_name" ] || [ "$package_name" = "null" ]; then
      echo "❌ Package '$package_id' not found"
      sleep 2
      return
    fi
  else
    echo "❌ jq is required but not installed"
    sleep 2
    return
  fi
  
  # Get deployment file path
  echo "📝 Preparing deployment configuration..."
  deploy_file=$(get_instance_file_path "$package_name" "$package_id" 2>&1 | tail -1)
  repository=$(get_oci_repository_path "$package_name" "$package_id")

  if [ -z "$deploy_file" ] || [ ! -f "$deploy_file" ]; then
    echo "❌ Deployment file not found for package '$package_name'"
    echo "   Expected: $deploy_file"
    sleep 2
    return
  fi
  echo "✅ Using deployment file: $deploy_file"
  
  # Update deployment file with device and package info
  sed -i "s|{{DEVICE_ID}}|$device_id|g" "$deploy_file" 2>/dev/null || true
  sed -i "s|{{PACKAGE_ID}}|$package_id|g" "$deploy_file" 2>/dev/null || true
  sed -i "s|{{REPOSITORY}}|$repository|g" "$deploy_file" 2>/dev/null || true
  
  # Open configuration in editor
  echo ""
  echo "📝 Opening parameter configuration override screen..."
  echo ""
  
  # Detect default editor
  EDITOR="${EDITOR:-${VISUAL:-vi}}"
  
  # Open file in editor
  $EDITOR "$deploy_file"
  
  echo ""
  read -p "Want to finally deploy it? (y/n): " confirm
  
  if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled"
    sleep 1
    return
  fi
  
  echo ""
  echo "🚀 Deploying package '$package_id' to device '$device_id'..."
  
  if check_maestro_cli; then
    if ${MAESTRO_CLI_PATH}/maestro wfm apply -f "$deploy_file" 2>&1 | grep -q "successfully\|created\|updated"; then
      echo "✅ Done!"
      echo ""
      echo "📋 Updated deployments:"
      ${MAESTRO_CLI_PATH}/maestro wfm list deployment
    else
      echo "❌ Failed to deploy instance"
    fi
  fi
  
  echo ""
  read -p "Press Enter to go back..."
}

enduser_delete_instance() {
  clear
  echo "🗑️  Delete Instance"
  echo "=================="
  echo ""
  
  if check_maestro_cli; then
    ${MAESTRO_CLI_PATH}/maestro wfm list deployment
  fi
  
  echo ""
  read -p "Enter the deployment/instance ID to delete: " instance_id
  
  if [ -z "$instance_id" ]; then
    echo "❌ Instance ID is required"
    sleep 2
    return
  fi
  
  echo ""
  read -p "Are you sure you want to delete instance '$instance_id'? (y/n): " confirm
  
  if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Deletion cancelled"
    sleep 1
    return
  fi
  
  echo ""
  echo "🗑️  Deleting instance '$instance_id'..."
  if check_maestro_cli; then
    if ${MAESTRO_CLI_PATH}/maestro wfm delete deployment "$instance_id" 2>&1 | grep -q "successfully\|deleted"; then
      echo "✅ Instance deleted successfully!"
      echo ""
      echo "📋 Updated deployments:"
      ${MAESTRO_CLI_PATH}/maestro wfm list deployment
    else
      echo "❌ Failed to delete instance"
    fi
  fi
  
  echo ""
  read -p "Press Enter to go back..."
}

show_enduser_menu() {
  while true; do
    clear
    echo "🖥️  End-User Menu (OT etc...)"
    echo "============================="
    echo ""
    echo "1) List App Packages"
    echo "2) List Devices"
    echo "3) List Deployments"
    echo "4) List All"
    echo "5) Deploy Instance"
    echo "6) Delete Instance"
    echo "7) Go Back"
    echo ""
    
    read -p "Enter choice [1-7]: " choice
    
    case $choice in
      1) enduser_list_packages ;;
      2) enduser_list_devices ;;
      3) enduser_list_deployments ;;
      4) enduser_list_all ;;
      5) enduser_deploy_instance ;;
      6) enduser_delete_instance ;;
      7) return ;;
      *) echo "⚠️ Invalid choice"; sleep 1 ;;
    esac
  done
}

# ----------------------------
# Main Menu Functions
# ----------------------------
show_main_menu() {
  clear
  echo "🎛️  WFM CLI Interactive Interface"
  echo "================================="
  echo ""
  echo "Choose a persona:"
  echo "1) 📦 App Supplier"
  echo "2) 🖥️  End-User (OT etc...)"
  echo "3) 🚪 Exit"
  echo ""
  
  read -p "Enter choice [1-3]: " choice
  
  case $choice in
    1) show_supplier_menu ;;
    2) show_enduser_menu ;;
    3) echo "👋 Goodbye!"; exit 0 ;;
    *) echo "⚠️ Invalid choice"; sleep 1 ;;
  esac
}

# ----------------------------
# Main Script Execution
# ----------------------------
main_loop() {
  install_basic_utilities
  
  # Ensure packages directory exists
  mkdir -p "$PACKAGES_DIR"
  
  while true; do
    show_main_menu
  done
}

# ----------------------------
# Command Line Interface
# ----------------------------
if [[ -z "$1" ]]; then
  # No arguments - run interactive mode
  main_loop
else
  # Command line mode for automation
  case "$1" in
    # Supplier commands
    supplier-upload)
      supplier_upload_package
      ;;
    supplier-list)
      supplier_list_packages
      ;;
    supplier-delete)
      supplier_delete_package
      ;;
    supplier-package-creation-wizard)
      supplier_package_creation_wizard
      ;;
    
    # End-user commands
    list-packages)
      enduser_list_packages
      ;;
    list-devices)
      enduser_list_devices
      ;;
    list-deployments)
      enduser_list_deployments
      ;;
    list-all)
      enduser_list_all
      ;;
    deploy)
      enduser_deploy_instance
      ;;
    delete-instance)
      enduser_delete_instance
      ;;
    
    # Help
    help|--help|-h)
      echo "WFM CLI - Workflow Management Command Line Interface"
      echo ""
      echo "Usage: $0 [command]"
      echo ""
      echo "Interactive Mode:"
      echo "  $0                    Run in interactive menu mode"
      echo ""
      echo "App Supplier Commands:"
      echo "  supplier-upload       Upload an app package"
      echo "  supplier-list         List all app packages"
      echo "  supplier-delete       Delete an app package"
      echo ""
      echo "End-User Commands:"
      echo "  list-packages         List available app packages"
      echo "  list-devices          List available devices"
      echo "  list-deployments      List current deployments"
      echo "  list-all              List all resources"
      echo "  deploy                Deploy an instance"
      echo "  delete-instance       Delete a deployment instance"
      echo ""
      echo "Other:"
      echo "  help, --help, -h      Show this help message"
      echo ""
      echo "Environment Variables:"
      echo "  EXPOSED_HARBOR_IP     Harbor registry IP (default: 127.0.0.1)"
      echo "  EXPOSED_HARBOR_PORT   Harbor registry port (default: 8081)"
      echo "  REGISTRY_USER         Registry username (default: admin)"
      echo "  REGISTRY_PASS         Registry password (default: Harbor12345)"
      echo "  OCI_ORGANIZATION      OCI organization (default: library)"
      echo "  MAESTRO_CLI_PATH      Path to maestro CLI (default: \$HOME/symphony/cli)"
      echo "  PACKAGES_DIR          App packages directory (default: \$HOME/sandbox/poc/tests/artefacts)"
      ;;
    
    *)
      echo "❌ Unknown command: $1"
      echo "Run '$0 help' for usage information"
      exit 1
      ;;
  esac
fi