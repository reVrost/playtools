## playtools

## Features

- Three primary actions:
  - Start a new sweepstake quest
  - Process sweepstake calculations (without distributing rewards)
  - Complete sweepstake quests (with reward distribution)
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

### Workflow

1. Select your environment (Development or Production)
2. Choose an action:
   - Start Sweepstake: Create a new sweepstake quest
   - Process Sweepstake: Calculate winners without distributing rewards
   - Complete Sweepstake: Calculate winners and distribute rewards
3. Provide required information:
   - For Start: Enter duration in minutes
   - For Process/Complete: Enter the sweepstake quest ID
4. View detailed Lambda execution results and logs

## Troubleshooting

### AWS SSO Session Issues

If your SSO session has expired, the tool will automatically attempt to reauthenticate. Follow the browser prompts to complete the login process.

### Lambda Invocation Failures

Check the detailed error messages and logs displayed in the output screen. Common issues include:

- Invalid sweepstake quest ID
- Insufficient permissions
- Network connectivity problems

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License
