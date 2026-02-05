#!/bin/bash
# ========================================================================
#                 ENVIRONMENT SETUP FUNCTIONS
#          Shared environment configuration logic
# ========================================================================

# Source directory for this script
SETUP_ENV_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load common functions if not already loaded
if [ -z "$COMMON_FUNCTIONS_LOADED" ]; then
    source "$SETUP_ENV_DIR/common.sh"
    COMMON_FUNCTIONS_LOADED=true
fi

# ========================================================================
#                    ENVIRONMENT TEMPLATE HANDLING
# ========================================================================

# Create environment file from template
create_env_from_template() {
    local template_source="${1:-$SETUP_ENV_DIR/.env.template}"
    local target_env="${2:-.env}"
    
    if [ ! -f "$template_source" ]; then
        print_error "Template file not found: $template_source"
        return 1
    fi
    
    cp "$template_source" "$target_env"
    convert_to_unix_line_endings "$target_env"
}

# Append missing variables to existing .env
append_missing_env_vars() {
    local template_source="${1:-$SETUP_ENV_DIR/.env.template}"
    local target_env="${2:-.env}"
    
    if [ ! -f "$template_source" ]; then
        print_error "Template file not found: $template_source"
        return 1
    fi
    
    if [ ! -f "$target_env" ]; then
        print_warning "$target_env does not exist. Creating from template."
        create_env_from_template "$template_source" "$target_env"
        return 0
    fi
    
    # Read existing .env content
    local existing_vars=$(grep -E '^[A-Z_]+=.*' "$target_env" 2>/dev/null | cut -d'=' -f1 | sort)
    
    # Read template variables
    local template_vars=$(grep -E '^[A-Z_]+=.*' "$template_source" | cut -d'=' -f1 | sort)
    
    # Find missing variables
    local missing_count=0
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Create temporary file for missing vars
    local temp_missing=$(mktemp)
    
    while IFS= read -r var_name; do
        if ! echo "$existing_vars" | grep -q "^${var_name}$"; then
            grep "^${var_name}=" "$template_source" >> "$temp_missing"
            ((missing_count++))
        fi
    done <<< "$template_vars"
    
    if [ $missing_count -gt 0 ]; then
        echo "" >> "$target_env"
        echo "# Added by setup script on $timestamp" >> "$target_env"
        cat "$temp_missing" >> "$target_env"
        print_step "Added $missing_count missing variable(s) to $target_env"
    else
        print_step "All variables already exist in $target_env"
    fi
    
    rm -f "$temp_missing"
    convert_to_unix_line_endings "$target_env"
    
    return 0
}

# Main environment setup function
setup_environment() {
    local template_source="${1:-$SETUP_ENV_DIR/.env.template}"
    local target_env="${2:-.env}"
    
    echo -e "\n${BLUE}Setting up environment...${NC}\n"
    
    if [ ! -f "$target_env" ]; then
        # If .env doesn't exist, create it from template
        create_env_from_template "$template_source" "$target_env"
        print_step "Created $target_env with default environment variables"
    else
        # If .env exists, append missing variables
        print_warning "$target_env already exists. Checking for missing variables..."
        append_missing_env_vars "$template_source" "$target_env"
    fi
}

# ========================================================================
#                    INTERACTIVE CONFIGURATION
# ========================================================================

# Configure public host interactively
configure_host_interactive() {
    local target_env="${1:-.env}"
    
    echo -e "\n${BLUE}Network Configuration${NC}\n"
    
    # Detect local IP for display purposes
    local local_ip=$(hostname -I 2>/dev/null | awk '{print $1}' || ipconfig getifaddr en0 2>/dev/null || echo "")
    
    if [ -n "$local_ip" ]; then
        echo "Detected local IP: $local_ip"
        echo ""
    fi
    
    echo "Enter your public IP address or domain name:"
    echo "(Examples: 192.168.1.100, myapp.example.com, or localhost for local development)"
    echo ""
    read -p "IP/Domain [localhost]: " PUBLIC_HOST
    
    # Use localhost as default if nothing entered
    if [ -z "$PUBLIC_HOST" ]; then
        PUBLIC_HOST="localhost"
    fi
    
    update_env_var "PUBLIC_HOST" "$PUBLIC_HOST" "$target_env"
    print_step "Configured PUBLIC_HOST=$PUBLIC_HOST"
}

# Configure owner registration interactively
configure_owner_interactive() {
    local target_env="${1:-.env}"
    
    echo -e "\n${BLUE}Owner Registration Configuration${NC}\n"
    
    echo "Enter owner registration details (press Enter to use defaults):"
    echo ""
    
    read -p "First Name [Admin]: " OWNER_FIRST_NAME
    if [ -z "$OWNER_FIRST_NAME" ]; then
        OWNER_FIRST_NAME="Admin"
    fi
    
    read -p "Last Name [User]: " OWNER_LAST_NAME
    if [ -z "$OWNER_LAST_NAME" ]; then
        OWNER_LAST_NAME="User"
    fi
    
    read -p "Email [admin@example.com]: " OWNER_EMAIL
    if [ -z "$OWNER_EMAIL" ]; then
        OWNER_EMAIL="admin@example.com"
    fi
    
    read -p "Password [Admin@123]: " OWNER_PASSWORD
    if [ -z "$OWNER_PASSWORD" ]; then
        OWNER_PASSWORD="Admin@123"
    fi
    
    # Update all owner configuration variables
    update_env_var "OWNER_FIRST_NAME" "$OWNER_FIRST_NAME" "$target_env"
    update_env_var "OWNER_LAST_NAME" "$OWNER_LAST_NAME" "$target_env"
    update_env_var "OWNER_EMAIL" "$OWNER_EMAIL" "$target_env"
    update_env_var "OWNER_PASSWORD" "$OWNER_PASSWORD" "$target_env"
    
    print_step "Owner configuration set"
}

# ========================================================================
#                    NON-INTERACTIVE CONFIGURATION
# ========================================================================

# Configure with default values (non-interactive)
configure_with_defaults() {
    local target_env="${1:-.env}"
    
    echo -e "\n${BLUE}========================================================================"
    echo "                      NETWORK CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    
    local PUBLIC_HOST="localhost"
    echo "Using default IP/domain: $PUBLIC_HOST"
    echo ""
    
    update_env_var "PUBLIC_HOST" "$PUBLIC_HOST" "$target_env"
    print_step "Configured PUBLIC_HOST=$PUBLIC_HOST"
    
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                   OWNER REGISTRATION CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    echo "Using default values:"
    echo ""
    
    local OWNER_FIRST_NAME="Admin"
    local OWNER_LAST_NAME="User"
    local OWNER_EMAIL="admin@example.com"
    local OWNER_PASSWORD="Admin@123"
    
    echo "   First Name: $OWNER_FIRST_NAME"
    echo "   Last Name:  $OWNER_LAST_NAME"
    echo "   Email:      $OWNER_EMAIL"
    echo "   Password:   $OWNER_PASSWORD"
    echo ""
    
    # Update all owner configuration variables
    update_env_var "OWNER_FIRST_NAME" "$OWNER_FIRST_NAME" "$target_env"
    update_env_var "OWNER_LAST_NAME" "$OWNER_LAST_NAME" "$target_env"
    update_env_var "OWNER_EMAIL" "$OWNER_EMAIL" "$target_env"
    update_env_var "OWNER_PASSWORD" "$OWNER_PASSWORD" "$target_env"
    
    print_step "Owner configuration set"
}
