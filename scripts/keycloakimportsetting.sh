#!/bin/bash

input_file="./dockerfiles/import/realm-import.sample.json"
output_file="./dockerfiles/import/realm-import.json"
env_file="./.env"

source "$env_file"

if grep -q "<<[^>]*>>" "$input_file"; then
    echo
    echo "## MC-IAM-MANAGER Init Setup ##"
    echo " - Please enter the changes. If not, use the environment variable."
    echo " - You can set Values in ./.mciammanager_init_env"
    echo
else
    echo "No variables to set found in the file."
    exit 1
fi

cp "$input_file" "$output_file"

update_env_file() {
    key=$1
    value=$2
    if grep -q "^$key=" "$env_file"; then
        sed -i "s|^$key=.*|$key=$value|" "$env_file"
    else
        echo "$key=$value" >> "$env_file"
    fi
}

vars=$(grep -o "<<[^>]*>>" "$input_file" | sort | uniq)

for var in $vars; do
    clean_var=$(echo "$var" | sed 's/<<\|>>//g')

    read -p "$clean_var  : " user_input

    if [ -z "$user_input" ]; then
        eval env_value=\$$clean_var
        if [ -z "$env_value" ]; then
            echo
            echo "No input provided, and environment variable $clean_var is not set. Exiting."
            exit 1
        else
            echo
            echo "No input provided. Using environment variable $clean_var."
            echo
            user_input="$env_value"
        fi
    else
        update_env_file "$clean_var" "$user_input"
    fi

    sed -i "s|$var|$user_input|g" "$output_file"
done

echo
echo "** File updated successfully and saved as $output_file. **"
echo "** Environment variables updated in $env_file. **"
