#!/bin/bash

# Load environment variables
source ./.env

# Global variables
force_mode=false
MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN=""
CURRENT_USER_ID=""
CURRENT_USER_PASSWORD=""

# API 엔드포인트 설정
API_ENDPOINT="$MCIAMMANAGER_HOST/api"

# Function to display usage information
usage() {
  echo "Usage: $0 [-f]"
  echo "  -f: Force mode (continue on errors)"
  exit 1
}

# Function to handle command-line options
parse_options() {
  while getopts "f" opt; do
    case $opt in
      f) force_mode=true ;;
      *) usage ;;
    esac
  done
  shift $((OPTIND - 1))
}

# Function to login
login() {
  echo "Logging in..."

  # Ask for user ID if not provided
  if [ -z "$CURRENT_USER_ID" ]; then
    read -p "Enter user ID (default: $MCIAMMANAGER_PLATFORMADMIN_ID): " user_id
    CURRENT_USER_ID="${user_id:-$MCIAMMANAGER_PLATFORMADMIN_ID}"
  fi

  # Ask for password if not provided
  if [ -z "$CURRENT_USER_PASSWORD" ]; then
    read -s -p "Enter password (default: $MCIAMMANAGER_PLATFORMADMIN_PASSWORD): " password
    echo
    CURRENT_USER_PASSWORD="${password:-$MCIAMMANAGER_PLATFORMADMIN_PASSWORD}"
  fi

  response=$(curl --location --silent --header 'Content-Type: application/json' --data "{
    \"id\":\"$CURRENT_USER_ID\",
    \"password\":\"$CURRENT_USER_PASSWORD\"
  }" "$API_ENDPOINT/auth/login")

  MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN=$(echo "$response" | jq -r '.access_token')
  if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
    echo "Login failed."
    $force_mode || exit 1
  else
    echo "Login successful."
  fi
}

# Function to create users from file
create_users() {
  local file="./add_demo_user.json"

  # Check if the file exists
  if [ ! -f "$file" ]; then
    echo "Error: File '$file' not found."
    $force_mode || exit 1
  fi

  echo "Creating users from file '$file'..."

  # Read JSON array from file
  users=$(jq -c '.[]' "$file")

  for user in $users; do
    local user_data=$(echo "$user")
    local user_id=$(echo "$user_data" | jq -r '.id')
    local password=$(echo "$user_data" | jq -r '.password')
    local first_name=$(echo "$user_data" | jq -r '.firstName')
    local last_name=$(echo "$user_data" | jq -r '.lastName')
    local email=$(echo "$user_data" | jq -r '.email')
    local description=$(echo "$user_data" | jq -r '.description')

    # Create user
    json_data=$(jq -n --arg id "$user_id" --arg password "$password" \
      --arg firstName "$first_name" --arg lastName "$last_name" \
      --arg email "$email" --arg description "$description" \
      '{id: $id, password: $password, firstName: $firstName, lastName: $lastName, email: $email, description: $description}')

    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
      --location "$API_ENDPOINT/user" \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
      --data "$json_data")

    if [ "$response" -ne 200 ]; then
      echo "Failed to create user $user_id"
      $force_mode || exit 1
    else
      echo "User created successfully: $user_id"
    fi

    # Activate user
    json_data=$(jq -n --arg userId "$user_id" '{userId: $userId}')

    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
      --location "$API_ENDPOINT/user/active" \
      --header 'Content-Type: application/json' \
      --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
      --data "$json_data")

    if [ "$response" -ne 200 ]; then
      echo "Failed to activate user $user_id"
      $force_mode || exit 1
    else
      echo "User activated successfully: $user_id"
    fi
  done
}

# Function definitions for new actions
menu_management() {
  while true; do
    echo
    echo "Menu Management:"
    echo "1. View menu list"
    echo "2. Add new menu"
    echo "3. Back to main menu"
    read -p "Enter your choice (1-3): " choice

    case "$choice" in
      1)
        echo "Viewing menu list..."
        curl -s -X GET "$API_ENDPOINT/menus" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" | jq
        ;;
      2)
        echo "Adding new menu..."
        curl -s -X POST "$API_ENDPOINT/menus" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
          -d '{"name": "Test Menu", "description": "Test Menu Description"}' | jq
        ;;
      3)
        echo "Returning to main menu..."
        return
        ;;
      *)
        echo "Invalid choice. Please enter 1, 2, or 3."
        ;;
    esac
  done
}

role_management() {
  while true; do
    echo
    echo "Role Management:"
    echo "1. View platform role list"
    echo "2. Add new platform role"
    echo "3. View workspace role list"
    echo "4. Add new workspace role"
    echo "5. Back to main menu"
    read -p "Enter your choice (1-5): " choice

    case "$choice" in
      1)
        echo "Viewing platform role list..."
        curl -s -X GET "$API_ENDPOINT/platform-roles" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" | jq
        ;;
      2)
        echo "Adding new platform role..."
        curl -s -X POST "$API_ENDPOINT/platform-roles" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
          -d '{"name": "Test Platform Role", "description": "Test Platform Role Description"}' | jq
        ;;
      3)
        echo "Viewing workspace role list..."
        curl -s -X GET "$API_ENDPOINT/workspace-roles" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" | jq
        ;;
      4)
        echo "Adding new workspace role..."
        curl -s -X POST "$API_ENDPOINT/workspace-roles" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
          -d '{"name": "Test Workspace Role", "description": "Test Workspace Role Description"}' | jq
        ;;
      5)
        echo "Returning to main menu..."
        return
        ;;
      *)
        echo "Invalid choice. Please enter a number between 1 and 5."
        ;;
    esac
  done
}

workspace_management() {
  while true; do
    echo
    echo "Workspace Management:"
    echo "1. View workspace list"
    echo "2. Add new workspace"
    echo "3. Back to main menu"
    read -p "Enter your choice (1-3): " choice

    case "$choice" in
      1)
        echo "Viewing workspace list..."
        curl -s -X GET "$API_ENDPOINT/workspaces" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" | jq
        ;;
      2)
        echo "Adding new workspace..."
        curl -s -X POST "$API_ENDPOINT/workspaces" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
          -d '{"name": "Test Workspace", "description": "Test Workspace Description"}' | jq
        ;;
      3)
        echo "Returning to main menu..."
        return
        ;;
      *)
        echo "Invalid choice. Please enter 1, 2, or 3."
        ;;
    esac
  done
}

user_management() {
  while true; do
    echo
    echo "User Management:"
    echo "1. View user list"
    echo "2. Add new user"
    echo "3. Login with new user"
    echo "4. Back to main menu"
    read -p "Enter your choice (1-4): " choice

    case "$choice" in
      1)
        echo "Viewing user list..."
        curl -s -X GET "$API_ENDPOINT/users" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" | jq
        ;;
      2)
        echo "Adding new user..."
        curl -s -X POST "$API_ENDPOINT/users" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
          -d '{"username": "testuser", "password": "testpassword", "email": "testuser@example.com"}' | jq
        ;;
      3)
        echo "Logging in with new user..."
        curl -s -X POST "$API_ENDPOINT/auth/login" \
          -H "Content-Type: application/json" \
          -d '{"username": "testuser", "password": "testpassword"}' | jq
        ;;
      4)
        echo "Returning to main menu..."
        return
        ;;
      *)
        echo "Invalid choice. Please enter a number between 1 and 4."
        ;;
    esac
  done
}

# Main script logic
main() {
  parse_options "$@"

  while true; do
    echo
    echo "Choose an action:"
    echo "1. Login"
    echo "2. Create users from file"
    echo "3. Menu management"
    echo "4. Role management"
    echo "5. Workspace management"
    echo "6. User management"
    echo "7. Exit"
    read -p "Enter your choice (1-7): " choice

    case "$choice" in
      1)
        login
        ;;
      2)
        if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
          echo "Error: You must log in first."
        else
          create_users
        fi
        ;;
      3)
        if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
          echo "Error: You must log in first."
        else
          menu_management
        fi
        ;;
      4)
        if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
          echo "Error: You must log in first."
        else
          role_management
        fi
        ;;
      5)
        if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
          echo "Error: You must log in first."
        else
          workspace_management
        fi
        ;;
      6)
        if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
          echo "Error: You must log in first."
        else
          user_management
        fi
        ;;
      7)
        echo "Exiting..."
        exit 0
        ;;
      *)
        echo "Invalid choice. Please enter a number between 1 and 7."
        ;;
    esac
  done
}

main
