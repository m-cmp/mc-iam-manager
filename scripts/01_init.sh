#!/bin/bash

# 01_init.sh - Initialize admin user
# 관리자 사용자 초기화 스크립트

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Load environment variables from .env file
if [[ -f "$PROJECT_ROOT/.env" ]]; then
  echo "Loading environment variables from .env file..."
  # Only export valid environment variable assignments (key=value format)
  while IFS= read -r line; do
    # Skip empty lines, comments, and lines without '='
    if [[ -n "$line" && ! "$line" =~ ^[[:space:]]*# && "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; then
      export "$line"
    fi
  done < "$PROJECT_ROOT/.env"
elif [[ -f "$PROJECT_ROOT/.env_sample" ]]; then
  echo "Warning: .env file not found, using .env_sample as reference..."
  echo "Please create .env file with proper values."
  # Only export valid environment variable assignments (key=value format)
  while IFS= read -r line; do
    # Skip empty lines, comments, and lines without '='
    if [[ -n "$line" && ! "$line" =~ ^[[:space:]]*# && "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; then
      export "$line"
    fi
  done < "$PROJECT_ROOT/.env_sample"
else
  echo "Error: Neither .env nor .env_sample file found in project root."
  exit 1
fi

# Default values for admin user (fallback if env vars are not set)
DEFAULT_EMAIL="${MCIAMMANAGER_PLATFORMADMIN_EMAIL:-admin@example.com}"
DEFAULT_PASSWORD="${MCIAMMANAGER_PLATFORMADMIN_PASSWORD:-admin123}"
DEFAULT_USERNAME="${MCIAMMANAGER_PLATFORMADMIN_ID:-admin}"

# Function to display usage
show_usage() {
  echo "Usage: $0 [OPTIONS]"
  echo ""
  echo "This script reads admin credentials from .env file:"
  echo "  - MCIAMMANAGER_PLATFORMADMIN_EMAIL (email)"
  echo "  - MCIAMMANAGER_PLATFORMADMIN_PASSWORD (password)"
  echo "  - MCIAMMANAGER_PLATFORMADMIN_ID (username)"
  echo ""
  echo "Options:"
  echo "  -e, --email EMAIL     Admin email (overrides .env value)"
  echo "  -p, --password PASS   Admin password (overrides .env value)"
  echo "  -u, --username USER   Admin username (overrides .env value)"
  echo "  -h, --help           Show this help message"
  echo ""
  echo "Current values from .env:"
  echo "  Email: $DEFAULT_EMAIL"
  echo "  Username: $DEFAULT_USERNAME"
  echo "  Password: [hidden]"
  echo ""
  echo "Example:"
  echo "  $0 -e admin@company.com -p securepass123 -u administrator"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -e|--email)
      EMAIL="$2"
      shift 2
      ;;
    -p|--password)
      PASSWORD="$2"
      shift 2
      ;;
    -u|--username)
      USERNAME="$2"
      shift 2
      ;;
    -h|--help)
      show_usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      show_usage
      exit 1
      ;;
  esac
done

# Set default values if not provided
EMAIL=${EMAIL:-$DEFAULT_EMAIL}
PASSWORD=${PASSWORD:-$DEFAULT_PASSWORD}
USERNAME=${USERNAME:-$DEFAULT_USERNAME}

# API endpoint
API_URL="${MCIAMMANAGER_HOST}/api/initial-admin"
echo "MCIAMMANAGER_HOST: $MCIAMMANAGER_HOST"
echo "PORT: $PORT"

echo "Initializing admin user..."
echo "Email: $EMAIL"
echo "Username: $USERNAME"
echo "API URL: $API_URL"
echo ""

# Make the API call
response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\",
    \"username\": \"$USERNAME\"
  }")

# Extract response body and status code
http_code=$(echo "$response" | tail -n1)
response_body=$(echo "$response" | head -n -1)

echo "Response Status: $http_code"
echo "Response Body:"
echo "$response_body"

# Check if the request was successful
if [[ $http_code -eq 200 ]] || [[ $http_code -eq 201 ]]; then
  echo ""
  echo "✅ Admin user initialized successfully!"
  exit 0
else
  echo ""
  echo "❌ Failed to initialize admin user. HTTP Status: $http_code"
  exit 1
fi 