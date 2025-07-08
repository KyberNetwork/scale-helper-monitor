package kyberswap

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Config represents KyberSwap configuration
type Config struct {
	APIBaseURL string
	ClientID   string
}

// Client handles communication with KyberSwap API
type Client struct {
	baseURL  string
	clientID string
	client   *http.Client
	logger   *logrus.Logger
}

// NewClient creates a new KyberSwap API client
func NewClient(config Config, timeout time.Duration, logger *logrus.Logger) *Client {
	return &Client{
		baseURL:  config.APIBaseURL,
		clientID: config.ClientID,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// GetRoute fetches a route from KyberSwap API
func (c *Client) GetRoute(chainName string, tokenIn, tokenOut, amount string, availableSources []string, includedSources []string) (*KyberSwapRouteEncodedData, *KyberSwapRoute, error) {
	routeURL := fmt.Sprintf("%s/%s/api/v1/routes", c.baseURL, chainName)

	params := url.Values{}
	params.Add("tokenIn", tokenIn)
	params.Add("tokenOut", tokenOut)
	params.Add("amountIn", amount)
	if len(includedSources) > 0 {
		// Check if we need to randomly pick sources
		sourcesString := strings.Join(includedSources, ",")
		if strings.Contains(sourcesString, "random") {
			// randomly pick sources from availableSources
			sourcesCount := rand.Intn(len(availableSources)) + 1
			selectedSources := make([]string, 0, sourcesCount)
			for _ = range sourcesCount {
				selectedSources = append(selectedSources, availableSources[rand.Intn(len(availableSources))])
			}
			sourcesString = strings.Join(selectedSources, ",")
		} else {
			// Pick only the sources that are in the available sources list
			availableSourcesString := strings.Join(availableSources, ",")
			validSources := []string{}
			for _, source := range includedSources {
				if strings.Contains(availableSourcesString, source) {
					validSources = append(validSources, source)
				}
			}
			sourcesString = strings.Join(validSources, ",")	
		}

			
		if len(sourcesString) > 0 {
			params.Add("includedSources", sourcesString)
		} else {
			return nil, nil, fmt.Errorf("no valid sources found")
		}
	}

	fullURL := fmt.Sprintf("%s?%s", routeURL, params.Encode())

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %v", err)
	}
	// Add headers
	req.Header.Set("X-Client-Id", c.clientID)

	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fetch route failed: TokenIn: %s TokenOut: %s Amount: %s Chain: %s Response: %s", tokenIn, tokenOut, amount, chainName, string(body))
	}

	// Parse response
	var apiResponse KyberSwapAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Check API response code
	if apiResponse.Code != 0 {
		c.logger.WithFields(logrus.Fields{
			"status_code": apiResponse.Code,
			"response":    apiResponse.Message,
			"url":         fullURL,
		}).Warn("KyberSwap API returned non-0 status")
		return nil, nil, fmt.Errorf("KyberSwap API returned non-0 status")
	}

	// Fetch route encoded data
	routeBuildURL := fmt.Sprintf("%s/%s/api/v1/route/build", c.baseURL, chainName)

	// Create the request body for route/build
	buildRequest := map[string]interface{}{
		"routeSummary":         apiResponse.Data.RouteSummary,
		"sender":               "0xdeAD00000000000000000000000000000000dEAd",
		"recipient":            "0xdeAD00000000000000000000000000000000dEAd",
		"slippageTolerance":    5000,
		"ignoreCappedSlippage": true,
	}

	buildRequestJSON, err := json.Marshal(buildRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal build request: %v", err)
	}

	req, err = http.NewRequest("POST", routeBuildURL, strings.NewReader(string(buildRequestJSON)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Client-Id", c.clientID)

	resp, err = c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Return the route data even if encoded data fails
		route := apiResponse.Data.RouteSummary
		route.RouterAddress = apiResponse.Data.RouterAddress

		return nil, &route, fmt.Errorf("failed to build route: TokenIn: %s TokenOut: %s Amount: %s Chain: %s Response: %s", tokenIn, tokenOut, amount, chainName, string(body))
	}

	var encodedDataResponse KyberSwapEncodedData
	if err := json.Unmarshal(body, &encodedDataResponse); err != nil {
		// Return the route data even if encoded data parsing fails
		route := apiResponse.Data.RouteSummary
		route.RouterAddress = apiResponse.Data.RouterAddress

		return nil, &route, fmt.Errorf("failed to parse encoded data response: %v", err)
	}

	// Create route from response and add router address
	route := apiResponse.Data.RouteSummary
	route.RouterAddress = apiResponse.Data.RouterAddress

	return &encodedDataResponse.Data, &route, nil
}
