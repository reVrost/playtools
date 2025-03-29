package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	functionName = "your-lambda-function-name"
	profile      = "platform-nonprod-engineer"
	region       = "us-east-1"
)

type Action string

const (
	// ActionProcess processes the calcualtion sweepstake quest but not the distribution of rewards
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
	SweepstakeQuestID int  `json:"sweepstake_quest_id"`
	BatchSize         *int `json:"batch_size,omitempty"`

	// DurationMinutes Optional fields for action = start
	DurationMinutes     *int             `json:"duration_minutes,omitempty"`
	SweepstakeOverrides *json.RawMessage `json:"sweepstake_overrides,omitempty"`
}

type lambdaResult struct {
	output []string
	err    error
}

func invokeLambdaCmd(input string) tea.Cmd {
	return func() tea.Msg {
		output := []string{}
		err := invokeLambda(&output)
		return lambdaResult{output: output, err: err}
	}
}

func invokeLambda(output *[]string) error {
	// AWS SSO session stuff
	if err := checkSSOSession(output); err != nil {
		return err
	}
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create Lambda client
	client := lambda.NewFromConfig(cfg)

	// Define and marshal payload
	payload := EventPayload{}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Invoke Lambda
	result, err := client.Invoke(context.Background(), &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payloadBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to invoke Lambda: %v", err)
	}

	*output = append(*output, "Lambda invocation successful!")
	*output = append(*output, fmt.Sprintf("Response: %s", string(result.Payload)))
	if result.FunctionError != nil {
		*output = append(*output, fmt.Sprintf("Function error: %s", *result.FunctionError))
	}

	return nil
}

func checkSSOSession(output *[]string) error {
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

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc, action string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list   list.Model
	choice string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.action
			}
			fmt.Println(m.choice)
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func main() {
	items := []list.Item{
		item{title: "Start Sweepstake", action: "start", desc: "Play new sweepstake, overriding existing ones. Why not?"},
		item{title: "Process Sweepstake", action: "process", desc: "Process sweepstake calculation without distributing rewards. "},
		item{title: "Complete Sweepstake", action: "complete", desc: "Complete sweepstake calculation and distribute rewards"},
	}

	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "Rewards Tools"

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
