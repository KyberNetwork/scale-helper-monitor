package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"scale-helper-monitor/internal/clients/kyberswap"
	"time"

	"github.com/sirupsen/logrus"
)

// MonitoringResult represents the result structure needed for alerts
// This is a temporary interface until we fully refactor
type MonitoringResult interface {
	GetChainName() string
	GetTokenIn() string
	GetTokenOut() string
	GetAmount() string
	GetNewAmount() string
	GetIsSuccess() bool
	GetError() string
	GetInputData() string
	GetReturnedData() string
	GetRoute() [][]kyberswap.KyberSwapSwap
	GetOriginalTenderlyURL() string
	GetScaledTenderlyURL() string
}

type Client struct {
	webhookURL string
	client     *http.Client
	logger     *logrus.Logger
}

func NewClient(webhookURL string, timeout time.Duration, logger *logrus.Logger) *Client {
	return &Client{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// SendAlert sends an alert to Slack when monitoring fails
func (c *Client) SendAlert(failures []MonitoringResult, totalTestCases int) error {
	if c.webhookURL == "" {
		c.logger.Warn("Slack webhook URL not configured, skipping alert")
		return nil
	}

	if len(failures) == 0 {
		return nil
	}

	// Create Slack message for multiple results
	message := c.createAlertMessage(failures, totalTestCases)

	// Marshal to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	// Send to Slack
	resp, err := c.client.Post(c.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) createAlertMessage(failures []MonitoringResult, totalTestCases int) *Message {
	failureCount := len(failures)
	// Convert to GMT+7 timezone
	loc, _ := time.LoadLocation("Asia/Bangkok") // GMT+7
	localTime := time.Now().In(loc)
	title := fmt.Sprintf("🚨 Scale Helper Monitor Alert - %d Failures - %s",
		failureCount, localTime.Format(time.RFC1123))

	// Create a summary attachment
	summaryFields := []Field{
		{
			Title: "Total Failures",
			Value: fmt.Sprintf("%d", failureCount),
			Short: true,
		},
		{
			Title: "Total Test Cases",
			Value: fmt.Sprintf("%d", totalTestCases),
			Short: true,
		},
		{
			Title: "Success Rate",
			Value: fmt.Sprintf("%.2f%%", float64(totalTestCases-failureCount)/float64(totalTestCases)*100),
			Short: true,
		},
		{
			Title: "Affected Chains",
			Value: c.formatChainList(c.getUniqueChains(failures)),
			Short: true,
		},
	}

	summaryAttachment := Attachment{
		Color:  "danger",
		Title:  "📊 Alert Summary",
		Fields: summaryFields,
	}

	attachments := []Attachment{summaryAttachment}

	// Create individual attachments for each failure
	for i, result := range failures {
		fields := c.createResultFields(result)

		attachment := Attachment{
			Color:  "danger",
			Title:  fmt.Sprintf("❌ Failure %d: %s", i+1, result.GetChainName()),
			Fields: fields,
		}
		attachments = append(attachments, attachment)
	}

	return &Message{
		Text:        title,
		Attachments: attachments,
	}
}

// createResultFields creates common fields for a monitoring result
func (c *Client) createResultFields(result MonitoringResult) []Field {
	fields := []Field{
		{
			Title: "Chain",
			Value: result.GetChainName(),
			Short: true,
		},
		{
			Title: "Token Addresses",
			Value: fmt.Sprintf("In: `%s`\nOut: `%s`", result.GetTokenIn(), result.GetTokenOut()),
			Short: false,
		},
		{
			Title: "Amount",
			Value: result.GetAmount(),
			Short: true,
		},
		{
			Title: "New Amount",
			Value: result.GetNewAmount(),
			Short: true,
		},
	}

	// Add Tenderly simulation links
	if originalURL := result.GetOriginalTenderlyURL(); originalURL != "" {
		fields = append(fields, Field{
			Title: "Original Swap Simulation",
			Value: fmt.Sprintf("<%s|🔗 View in Tenderly>", originalURL),
			Short: true,
		})
	}

	if scaledURL := result.GetScaledTenderlyURL(); scaledURL != "" {
		fields = append(fields, Field{
			Title: "Scaled Swap Simulation",
			Value: fmt.Sprintf("<%s|🔗 View Failed Simulation>", scaledURL),
			Short: true,
		})
	}

	// Add error field if there's an error
	if result.GetError() != "" {
		fields = append(fields, Field{
			Title: "Error",
			Value: fmt.Sprintf("```%s```", result.GetError()),
			Short: false,
		})
	}

	// Add detailed route sequence information with pool types
	route := result.GetRoute()
	if len(route) > 0 {
		routeInfo := fmt.Sprintf("```Sequence Steps: %d\n", len(route))

		for i, sequence := range route {
			routeInfo += fmt.Sprintf("\n--- Step %d ---\n", i+1)

			if len(sequence) > 0 {
				for j, swap := range sequence {
					routeInfo += fmt.Sprintf("Pool %d:\n", j+1)
					routeInfo += fmt.Sprintf("  Type: %s\n", swap.PoolType)
					routeInfo += fmt.Sprintf("  Exchange: %s\n", swap.Exchange)
					if j < len(sequence)-1 {
						routeInfo += "\n"
					}
				}
			}
		}
		routeInfo += "```"

		fields = append(fields, Field{
			Title: "Sequence Details",
			Value: routeInfo,
			Short: false,
		})
	}

	return fields
}

// getUniqueChains extracts unique chain names from results
func (c *Client) getUniqueChains(results []MonitoringResult) []string {
	chainMap := make(map[string]bool)
	for _, result := range results {
		chainMap[result.GetChainName()] = true
	}

	chains := make([]string, 0, len(chainMap))
	for chain := range chainMap {
		chains = append(chains, chain)
	}
	return chains
}

// formatChainList formats a list of chains for display
func (c *Client) formatChainList(chains []string) string {
	if len(chains) == 0 {
		return "None"
	}
	if len(chains) == 1 {
		return chains[0]
	}
	if len(chains) <= 3 {
		result := ""
		for i, chain := range chains {
			if i > 0 {
				result += ", "
			}
			result += chain
		}
		return result
	}
	return fmt.Sprintf("%s, %s, and %d more", chains[0], chains[1], len(chains)-2)
}
