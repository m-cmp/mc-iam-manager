#!/bin/bash

source ../../.env

login() {
    read -p "Enter the platformadmin ID: " MCIAMMANAGER_PLATFORMADMIN_ID
    read -s -p "Enter the platformadmin password: " MCIAMMANAGER_PLATFORMADMIN_PASSWORD
    echo
    response=$(curl --location --silent --header 'Content-Type: application/json' --data '{
        "id":"'"$MCIAMMANAGER_PLATFORMADMIN_ID"'",
        "password":"'"$MCIAMMANAGER_PLATFORMADMIN_PASSWORD"'"
    }' "$MCIAMMANAGER_HOST/api/auth/login")
    MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN="$(echo "$response" | jq -r '.access_token')"
    echo "Login response: $response"
    echo "Login successful"
}

register_menu() {
    echo "Registering menu data..."
    wget -q -O ./menu.yaml "$MCWEBCONSOLE_MENUYAML"
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MCIAMMANAGER_HOST/api/setup/menus/register-from-yaml")
    echo "Menu registration response: $response"
    echo "Menu registration completed"
}

register_workspace() {
    read -p "Enter workspace name: " workspace_name
    json_data=$(jq -n --arg name "$workspace_name" \
        '{name: $name}')
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$MCIAMMANAGER_HOST/api/workspaces/")
    echo "Workspace registration response: $response"
    echo "Workspace registration completed"
}

sync_projects() {
    echo "Syncing projects with mc-infra-manager..."
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        "$MCIAMMANAGER_HOST/api/projects/sync")
    echo "Project sync response: $response"
    echo "Project sync completed"
}

map_workspace_projects() {
    read -p "Enter workspace ID: " workspace_id
    json_data=$(jq -n --arg workspace_id "$workspace_id" --arg all_projects "true" \
        '{workspace_id: $workspace_id, all_projects: $all_projects}')
    response=$(curl -s -X POST \
        --header "Authorization: Bearer $MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        "$MCIAMMANAGER_HOST/api/workspaces/projects/mapping")
    echo "Workspace-Project mapping response: $response"
    echo "Workspace-Project mapping completed"
}

while true; do
    echo "Select an option:"
    echo "0. Exit"
    echo "1. PlatformAdmin Login"
    echo "2. Register Menu"
    echo "3. Register Workspace"
    echo "4. Sync Projects"
    echo "5. Map Workspace-All Projects"
    
    read -p "Enter your choice (0-5): " choice
    
    case $choice in
        0)
            echo "Exiting..."
            exit 0
            ;;
        1)
            login
            ;;
        2)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                register_menu
            fi
            ;;
        3)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                register_workspace
            fi
            ;;
        4)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                sync_projects
            fi
            ;;
        5)
            if [ -z "$MCIAMMANAGER_PLATFORMADMIN_ACCESSTOKEN" ]; then
                echo "Please login first (option 1)"
            else
                map_workspace_projects
            fi
            ;;
        *)
            echo "Invalid option. Please try again."
            ;;
    esac
    
    echo
done 