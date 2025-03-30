## playtools

## Features

- Three primary actions:
  - Start a new ss quest
  - Process ss calculations 
  - Complete ss quests 
- AWS SSO authentication handling
- Lambda function invocation with detailed response and logs display

## Prerequisites

- Go 1.18 or later
- AWS CLI installed and configured
- AWS SSO profiles set up for your environments

## Installation

### Using Go

```bash
go install github.com/revrost/playtools
```

## Configuration

Before using the tool, make sure your AWS SSO profiles are properly configured in your `~/.aws/config` file:

```
[profile platform-nonprod-engineer]
sso_start_url = https://your-sso-portal.awsapps.com/start
sso_region = your-sso-region
sso_account_id = your-dev-account-id
sso_role_name = YourSSORoleName
region = your-aws-region

[profile platform-prod-engineer]
sso_start_url = https://your-sso-portal.awsapps.com/start
sso_region = your-sso-region
sso_account_id = your-prod-account-id
sso_role_name = YourSSORoleName
region = your-aws-region
```

## Usage

Installed with `go install`:

```bash
playtools
```

### Navigation

- Use arrow keys (↑/↓) to navigate through options
- Press Enter to select an option
- Press 'b' to go back to the previous screen
- Press 'q' or Ctrl+C to quit the application

### AWS SSO Session Issues

If your SSO session has expired, the tool will automatically attempt to reauthenticate. Follow the browser prompts to complete the login process.


