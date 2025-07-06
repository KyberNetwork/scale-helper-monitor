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
func (c *Client) SendAlert(result MonitoringResult) error {
	if c.webhookURL == "" {
		c.logger.Warn("Slack webhook URL not configured, skipping alert")
		return nil
	}

	// Create Slack message
	message := c.createAlertMessage(result)

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

	c.logger.WithFields(logrus.Fields{
		"chain":     result.GetChainName(),
		"tokenIn":   result.GetTokenIn(),
		"tokenOut":  result.GetTokenOut(),
		"amount":    result.GetAmount(),
		"newAmount": result.GetNewAmount(),
	}).Info("Successfully sent Slack alert")

	return nil
}

func (c *Client) createAlertMessage(result MonitoringResult) *Message {
	title := "🚨 Scale Helper Monitor Alert"

	color := "danger" // Red for failures

	// Create fields
	fields := []Field{
		{
			Title: "Chain",
			Value: fmt.Sprintf("%s", result.GetChainName()),
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
	route := result.GetRoute()

	for i, swap := range route {
		fields = append(fields, Field{
			Title: fmt.Sprintf("Pool %d", i+1),
			Value: fmt.Sprintf("```Pool Type: %s \n Exchange: %s \n TokenIn: %s \n TokenOut: %s ", swap[0].PoolType, swap[0].Exchange, swap[0].TokenIn, swap[0].TokenOut),
			Short: false,
		})
	}

	// Create attachment
	attachment := Attachment{
		Color:  color,
		Title:  fmt.Sprintf("Scaled swap simulation failed on %s", result.GetChainName()),
		Fields: fields,
	}

	// Create message
	message := &Message{
		Text:        title,
		Attachments: []Attachment{attachment},
	}

	return message
}
