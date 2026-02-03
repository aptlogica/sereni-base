#!/bin/bash
# Bash script for interactive owner configuration setup
# This script prompts the user for owner registration details and updates .env file
# Usage: bash setup-owner.sh       (interactive mode)
#        bash setup-owner.sh -y    (use defaults without prompts)

# Check if -y flag is provided
USE_DEFAULTS=false
if [ "$1" = "-y" ]; then
    USE_DEFAULTS=true
fi

# Function to get current value from .env file
get_env_value() {
    local key=$1
    if [ -f ".env" ]; then
        # Handle Windows CRLF line endings by removing \r
        local value=$(grep "^${key}=" .env | cut -d'=' -f2- | tr -d '\r')
        echo "$value"
    fi
}

# Function to update .env file
update_env_value() {
    local key=$1
    local value=$2
    local envFile=".env"
    
    if [ -f "$envFile" ]; then
        if grep -q "^${key}=" "$envFile"; then
            # Update existing key
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS
                sed -i '' "s/^${key}=.*/${key}=${value}/" "$envFile"
            else
                # Linux
                sed -i "s/^${key}=.*/${key}=${value}/" "$envFile"
            fi
        else
            # Add new key if it doesn't exist
            echo "${key}=${value}" >> "$envFile"
        fi
    fi
}

# Get current values
currentFirstName=$(get_env_value "OWNER_FIRST_NAME")
if [ -z "$currentFirstName" ]; then
    currentFirstName="Admin"
fi

currentLastName=$(get_env_value "OWNER_LAST_NAME")
if [ -z "$currentLastName" ]; then
    currentLastName="User"
fi

currentEmail=$(get_env_value "OWNER_EMAIL")
if [ -z "$currentEmail" ]; then
    currentEmail="admin@example.com"
fi

currentPassword=$(get_env_value "OWNER_PASSWORD")
if [ -z "$currentPassword" ]; then
    currentPassword="Admin@123"
fi

# Set values based on mode
if [ "$USE_DEFAULTS" = true ]; then
    firstName="Admin"
    lastName="User"
    email="admin@example.com"
    password="Admin@123"
    isDefaultMode=true
else
    # Prompt for new values
    read -p "First Name (current: $currentFirstName): " firstName
    if [ -z "$firstName" ]; then
        firstName="$currentFirstName"
    fi

    read -p "Last Name (current: $currentLastName): " lastName
    if [ -z "$lastName" ]; then
        lastName="$currentLastName"
    fi

    read -p "Email (current: $currentEmail): " email
    if [ -z "$email" ]; then
        email="$currentEmail"
    fi

    read -p "Password (current: $currentPassword): " password
    if [ -z "$password" ]; then
        password="$currentPassword"
    fi
    isDefaultMode=false
fi

# Update .env file
echo ""
echo "Updating .env file..."

update_env_value "OWNER_FIRST_NAME" "$firstName"
update_env_value "OWNER_LAST_NAME" "$lastName"
update_env_value "OWNER_EMAIL" "$email"
update_env_value "OWNER_PASSWORD" "$password"

echo ""
echo "========================================================================"
if [ "$isDefaultMode" = true ]; then
    echo "          Owner Configuration Set to Defaults!"
else
    echo "          Owner Configuration Updated Successfully!"
fi
echo "========================================================================"
echo ""
echo "First Name: $firstName"
echo "Last Name: $lastName"
echo "Email: $email"
echo "Password: $password"
echo ""
echo "========================================================================"
