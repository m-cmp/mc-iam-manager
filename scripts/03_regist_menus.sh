#!/bin/bash

# 03_regist_menus.sh - Register initial menus
# 초기 메뉴 등록 스크립트

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

# Function to display usage
show_usage() {
  echo "Usage: $0 [OPTIONS]"
  echo ""
  echo "This script registers initial menus via API call."
  echo "First logs in to get access token, then calls the API."
  echo ""
  echo "Options:"
  echo "  -h, --help           Show this help message"
  echo ""
  echo "Example:"
  echo "  $0"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
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

# Fix MCIAMMANAGER_HOST if it contains ${PORT} variable
if [[ "$MCIAMMANAGER_HOST" == *"\${PORT}"* ]]; then
  MCIAMMANAGER_HOST="http://localhost:${PORT}"
fi

# Get admin credentials from environment variables
ADMIN_USERNAME="${MCIAMMANAGER_PLATFORMADMIN_ID:-admin}"
ADMIN_PASSWORD="${MCIAMMANAGER_PLATFORMADMIN_PASSWORD:-admin123}"

echo "MCIAMMANAGER_HOST: $MCIAMMANAGER_HOST"

echo "Logging in to get access token..."
echo "Username: $ADMIN_USERNAME"
echo ""

# Login to get access token
LOGIN_URL="${MCIAMMANAGER_HOST}/api/auth/login"
login_response=$(curl -s -w "\n%{http_code}" -X POST "$LOGIN_URL" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"$ADMIN_USERNAME\",
    \"password\": \"$ADMIN_PASSWORD\"
  }")

# Extract login response body and status code
login_http_code=$(echo "$login_response" | tail -n1)
login_response_body=$(echo "$login_response" | head -n -1)

echo "Login Response Status: $login_http_code"
echo "Login Response Body:"
echo "$login_response_body"
echo ""

# Check if login was successful
if [[ $login_http_code -ne 200 ]]; then
  echo "❌ Login failed. HTTP Status: $login_http_code"
  exit 1
fi

# Extract access token from response (JSON response with access_token field)
# Try multiple methods to extract the token
ACCESS_TOKEN=$(echo "$login_response_body" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# If the above method fails, try alternative approach
if [[ -z "$ACCESS_TOKEN" ]]; then
  ACCESS_TOKEN=$(echo "$login_response_body" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
fi

# If still empty, try with jq if available
if [[ -z "$ACCESS_TOKEN" ]] && command -v jq &> /dev/null; then
  ACCESS_TOKEN=$(echo "$login_response_body" | jq -r '.access_token')
fi

if [[ -z "$ACCESS_TOKEN" ]]; then
  echo "❌ Failed to extract access token from login response"
  echo "Response body: $login_response_body"
  exit 1
fi

echo "Access token extracted successfully"

echo "✅ Login successful. Access token obtained."
echo ""

# API endpoint
API_URL="${MCIAMMANAGER_HOST}/api/setup/initial-menu"

echo "Registering initial menus..."
echo "API URL: $API_URL"
echo ""

# Make the API call with bearer token
response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN")

# Extract response body and status code
http_code=$(echo "$response" | tail -n1)
response_body=$(echo "$response" | head -n -1)

echo "Response Status: $http_code"
echo "Response Body:"
echo "$response_body"

# Check if the request was successful
if [[ $http_code -eq 200 ]] || [[ $http_code -eq 201 ]]; then
  echo ""
  echo "✅ Initial menus registration completed successfully!"
  exit 0
else
  echo ""
  echo "❌ Failed to register initial menus. HTTP Status: $http_code"
  exit 1
fi 