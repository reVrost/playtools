package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Constants
const (
	devEnv  = "dev"
	prodEnv = "prod"
)

// Profile mapping
var profileMap = map[string]string{
	devEnv:  "platform-nonprod-engineer",
	prodEnv: "platform-prod-engineer", // Adjust this if your prod profile is different
}

const sweepstakeFunctionName = "imx-rewards-%s-sweepstake-rewards-calculator"

type Action string

const (
	// ActionProcess processes the calculation sweepstake quest but not the distribution of rewards
	ActionProcess Action = "process"
	// ActionComplete completes a sweepstake quest and distributes rewards
	ActionComplete Action = "complete"
	// ActionStart starts a new sweepstake quest
	ActionStart Action = "start"
)

// EventPayload is the payload request for the lambda function
type EventPayload struct {
	Action Action `json:"action"`

	// DryRun is only applicable for process action
	DryRun            bool `json:"dry_run"`
	SweepstakeQuestID *int `json:"sweepstake_quest_id"`
	BatchSize         *int `json:"batch_size,omitempty"`

	// DurationMinutes Optional fields for action = start
	DurationMinutes     *int             `json:"duration_minutes,omitempty"`
	SweepstakeOverrides *json.RawMessage `json:"sweepstake_overrides,omitempty"`
}

// Screen types to track the current state
type Screen int

const (
	EnvironmentScreen Screen = iota
	ActionScreen
	LoadingScreen
	OutputScreen
	PromptScreen
)

// Messages
type lambdaResult struct {
	output []string
	logs   string
	err    error
}

type tickMsg time.Time

type item struct {
	title, desc, action string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Model struct that holds all application state
type model struct {
	currentScreen  Screen
	promptMessage  string
	promptQuestion string
	promptInput    textinput.Model
	envList        list.Model
	actionList     list.Model
	spinner        spinner.Model
	selectedEnv    string
	selectedAction string
	lambdaOutput   []string
	lambdaLogs     string
	lambdaErr      error
	width, height  int
}

func initialModel() model {
	// Environment selection items
	envItems := []list.Item{
		item{title: "Development", desc: "Use development environment", action: devEnv},
		item{title: "Production", desc: "Use production environment", action: prodEnv},
	}

	// Action selection items
	actionItems := []list.Item{
		item{title: "Start Sweepstake", desc: "Play new sweepstake, overriding existing ones", action: string(ActionStart)},
		item{title: "Process Sweepstake", desc: "Process sweepstake calculation without distributing rewards", action: string(ActionProcess)},
		item{title: "Complete Sweepstake", desc: "Complete sweepstake calculation and distribute rewards", action: string(ActionComplete)},
	}

	envList := list.New(envItems, list.NewDefaultDelegate(), 0, 0)
	envList.Title = "Select Environment"

	actionList := list.New(actionItems, list.NewDefaultDelegate(), 0, 0)
	actionList.Title = "Rewards Tools"

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize text input for prompt
	ti := textinput.New()
	ti.Placeholder = "Enter answer"
	ti.Focus()
	ti.CharLimit = 10
	ti.Width = 20

	return model{
		currentScreen: EnvironmentScreen,
		envList:       envList,
		actionList:    actionList,
		promptInput:   ti,
		spinner:       s,
		lambdaOutput:  []string{},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keyboard input during loading
		if m.currentScreen == LoadingScreen {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "b":
			if m.currentScreen == OutputScreen {
				m.currentScreen = ActionScreen
				return m, nil
			}

		case "enter":
			switch m.currentScreen {
			case EnvironmentScreen:
				if i, ok := m.envList.SelectedItem().(item); ok {
					m.selectedEnv = i.action
					m.currentScreen = ActionScreen
				}
				return m, nil

			case ActionScreen:
				if i, ok := m.actionList.SelectedItem().(item); ok {
					m.selectedAction = i.action
				} else {
					return m, nil
				}
				m.currentScreen = PromptScreen
				m.promptMessage = ""
				if m.selectedAction == string(ActionProcess) || m.selectedAction == string(ActionComplete) {
					m.promptQuestion = "Please enter sweepstake quest ID"
				} else {
					m.promptQuestion = "Please enter sweepstake duration in minutes"
				}
				m.promptInput.SetValue("")
				return m, textinput.Blink

			case PromptScreen:
				// Validate input is a number
				idStr := m.promptInput.Value()
				id, err := strconv.Atoi(idStr)
				if err != nil || id <= 0 {
					m.promptMessage = "Please enter a valid positive number"
					return m, nil
				}

				var payload EventPayload
				if m.selectedAction == string(ActionComplete) || m.selectedAction == string(ActionProcess) {

					payload = EventPayload{
						Action:            Action(m.selectedAction),
						SweepstakeQuestID: &id,
					}
				} else {
					// Start sweepstake action, with duration minutes
					payload = EventPayload{
						Action:          Action(m.selectedAction),
						DurationMinutes: &id,
					}
				}

				m.currentScreen = LoadingScreen
				return m, tea.Batch(
					m.spinner.Tick,
					invokeLambdaCmd(m.selectedEnv, payload),
				)
			}
		}

	case tickMsg:
		return m, nil

	case lambdaResult:
		m.lambdaOutput = msg.output
		m.lambdaLogs = msg.logs
		m.lambdaErr = msg.err
		m.currentScreen = OutputScreen
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := docStyle.GetFrameSize()
		m.envList.SetSize(msg.Width-h, msg.Height-v)
		m.actionList.SetSize(msg.Width-h, msg.Height-v)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update the appropriate list based on current screen
	var cmd tea.Cmd
	switch m.currentScreen {
	case EnvironmentScreen:
		m.envList, cmd = m.envList.Update(msg)
	case ActionScreen:
		m.actionList, cmd = m.actionList.Update(msg)
	case PromptScreen:
		m.promptInput, cmd = m.promptInput.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	switch m.currentScreen {
	case EnvironmentScreen:
		return docStyle.Render(m.envList.View())

	case ActionScreen:
		return docStyle.Render(m.actionList.View())

	case PromptScreen:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("\n\n  %s %s:\n\n", m.promptQuestion, m.selectedAction))
		sb.WriteString("  " + m.promptInput.View() + "\n\n")

		if m.promptMessage != "" {
			sb.WriteString("  " + m.promptMessage + "\n\n")
		}

		sb.WriteString("  Press Enter to continue or 'b' to go back\n")
		return docStyle.Render(sb.String())

	case LoadingScreen:
		return docStyle.Render(fmt.Sprintf("\n\n  %s Invoking Lambda in %s environment with action %s...\n\n  Please wait, this may take a few moments...",
			m.spinner.View(),
			m.selectedEnv,
			m.selectedAction))

	case OutputScreen:
		var output string
		if m.lambdaErr != nil {
			output = fmt.Sprintf("Error: %v\n\n", m.lambdaErr)
		} else {
			output = "Lambda Execution Summary:\n\n"
		}

		for _, line := range m.lambdaOutput {
			// Wrap long output lines
			output += lipgloss.NewStyle().Width(m.width-4).Render(line) + "\n"
		}

		if m.lambdaLogs != "" {
			output += "\n--- Lambda Logs ---\n\n"
			// Wrap the logs with appropriate width
			output += lipgloss.NewStyle().Width(m.width - 4).Render(m.lambdaLogs)
		}

		return docStyle.Render(fmt.Sprintf("%s\n\nPress 'b' to go back or 'q' to quit", output))
	}

	return "Loading..."
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func invokeLambdaCmd(env string, payload EventPayload) tea.Cmd {
	return func() tea.Msg {
		output := []string{}
		var logs string
		err := invokeLambda(env, payload, &output, &logs)
		return lambdaResult{output: output, logs: logs, err: err}
	}
}

func invokeLambda(env string, payload EventPayload, output *[]string, logs *string) error {
	profile := profileMap[env]
	functionName := fmt.Sprintf(sweepstakeFunctionName, env)

	*output = append(*output, fmt.Sprintf("Environment: %s", env))
	jsonPayload, _ := json.MarshalIndent(payload, "", "  ")
	*output = append(*output, fmt.Sprintf("Payload: %s", jsonPayload))

	// AWS SSO session check
	if err := checkSSOSession(profile, output); err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create Lambda client
	client := lambda.NewFromConfig(cfg)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Invoke Lambda with logs enabled
	result, err := client.Invoke(context.Background(), &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payloadBytes,
		LogType:      "Tail", // This will return the last 4KB of logs
	})
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %v", err)
	}

	*output = append(*output, "Lambda invocation successful!")

	// Process response
	var responseObj map[string]interface{}
	if err := json.Unmarshal(result.Payload, &responseObj); err != nil {
		*output = append(*output, fmt.Sprintf("Raw response: %s", string(result.Payload)))
	} else {
		formattedResponse, _ := json.MarshalIndent(responseObj, "", "  ")
		*output = append(*output, fmt.Sprintf("Response: %s", string(formattedResponse)))
	}

	// Check for function errors
	if result.FunctionError != nil {
		*output = append(*output, fmt.Sprintf("Function error: %s", *result.FunctionError))
	}

	// Decode and add logs if available
	if result.LogResult != nil {
		decodedLogs, err := decodeBase64(*result.LogResult)
		if err != nil {
			*output = append(*output, fmt.Sprintf("Error decoding logs: %v", err))
		} else {
			*logs = decodedLogs
		}
	}

	return nil
}

func decodeBase64(encoded string) (string, error) {
	// AWS Go SDK already decodes the base64 for us in LogResult
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %v", err)
	}
	return string(decoded), nil
}

func checkSSOSession(profile string, output *[]string) error {
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("AWS CLI not found")
	}

	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile)
	if err := cmd.Run(); err != nil {
		*output = append(*output, "SSO session expired. Logging in...")
		loginCmd := exec.Command("aws", "sso", "login", "--profile", profile)
		out, err := loginCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("SSO login failed: %v\nOutput: %s", err, out)
		}
		*output = append(*output, "SSO login successful")
	}
	return nil
}

// fetchLambdaLogs fetches recent logs using AWS CLI
func fetchLambdaLogs(profile, functionName string) (string, error) {
	// This is a simplified version, we're using the logs returned by the Lambda invocation
	// If you need more detailed logs, you would use the AWS CLI or CloudWatch Logs API here
	cmd := exec.Command("aws", "logs", "tail", fmt.Sprintf("/aws/lambda/%s", functionName), "--profile", profile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to fetch logs: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
