#!/bin/bash

input_file="realm-import.sample.json"
output_file="realm-import.json"

source ./.mciammanager_init_env

if grep -q "<<[^>]*>>" "$input_file"; then
    echo
    echo "## MC-IAM-MANAGER Init Setup ##"
    echo " - Please enter the changes. if not, use the environment variable."
    echo " - You can set Values in ./.mciammanager_init_env"
    echo
else
  echo "No variables to set found in the file."
  exit 1
fi

cp "$input_file" "$output_file"

vars=$(grep -o "<<[^>]*>>" "$input_file")

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
  fi

  sed -i "s|$var|$user_input|g" "$output_file"
#   echo "Updated $var to $user_input"
done

echo "File updated successfully and saved as $output_file."
