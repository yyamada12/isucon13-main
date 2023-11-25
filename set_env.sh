#!/bin/bash

# Interactive script to set environment variables and update .bash_profile

# Add a comment and empty line at the beginning of .bash_profile
echo -e "\n# env vars for isucon" >> ~/.bash_profile

# Function to prompt user for input with default value
prompt_for_input() {
    read -p "$1 (default: $2): " -r value
    if [ -z "$value" ]; then
        value="$2"
    fi
    echo "$value"
}

# Prompt for APP_DIR and update .bash_profile
default_app_dir="/home/isucon/webapp/go"
app_dir=$(prompt_for_input "Enter the value for APP_DIR" "$default_app_dir")
echo "export APP_DIR=\"$app_dir\"" >> ~/.bash_profile

# Prompt for APP_BUILD_CMD and update .bash_profile
default_app_build_cmd="make"
app_build_cmd=$(prompt_for_input "Enter the value for APP_BUILD_CMD" "$default_app_build_cmd")
echo "export APP_BUILD_CMD=\"$app_build_cmd\"" >> ~/.bash_profile

# Prompt for APP_SERVICE_NAME and update .bash_profile
default_app_service_name="isucon-go"
app_service_name=$(prompt_for_input "Enter the value for APP_SERVICE_NAME" "$default_app_service_name")
echo "export APP_SERVICE_NAME=\"$app_service_name\"" >> ~/.bash_profile

# Prompt for SLACK_TOKEN and update .bash_profile
default_slack_token="enter Slack User OAuth Token"
slack_token=$(prompt_for_input "Enter the value for SLACK_TOKEN https://api.slack.com/apps/A05GTSY2MKJ/oauth" "$default_slack_token")
echo "export SLACK_TOKEN=\"$slack_token\"" >> ~/.bash_profile

echo "Environment variables updated."
