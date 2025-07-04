package tenderly

import "net/http"

// Client represents a Tenderly API client
type Client struct {
	accessKey string
	username  string
	project   string
	baseURL   string
	client    *http.Client
}

// SimulationRequest represents a Tenderly simulation request
type SimulationRequest struct {
	NetworkID    string                 `json:"network_id"`
	From         string                 `json:"from"`
	To           string                 `json:"to"`
	Input        string                 `json:"input"`
	Value        string                 `json:"value,omitempty"`
	GasLimit     int64                  `json:"gas,omitempty"`
	GasPrice     string                 `json:"gas_price,omitempty"`
	Save         bool                   `json:"save"`
	SaveIfFails  bool                   `json:"save_if_fails"`
	SimulationType string               `json:"simulation_type,omitempty"`
	StateObjects map[string]interface{} `json:"state_objects,omitempty"`
}

// SimulationBundleRequest represents the bundle request structure
type SimulationBundleRequest struct {
	Simulations []SimulationRequest `json:"simulations"`
}

// SimulationBundleResponse represents the bundle response structure
type SimulationBundleResponse struct {
	SimulationResults []struct {
		Transaction *struct {
			Hash     string `json:"hash"`
			GasUsed  int64  `json:"gas_used"`
			Status   bool   `json:"status"`
		} `json:"transaction"`
		Simulation struct {
			ID           string `json:"id"`
			Status       bool   `json:"status"`
			ErrorMessage string `json:"error_message,omitempty"`
		} `json:"simulation"`
	} `json:"simulation_results"`
}



// SimulationResponse represents a Tenderly simulation response
type SimulationResponse struct {
	Simulation struct {
		ID          string `json:"id"`
		Status      bool   `json:"status"`
		ErrorInfo   *struct {
			ErrorMessage string `json:"error_message"`
			Address      string `json:"address"`
		} `json:"error_info,omitempty"`
		GasUsed int64  `json:"gas_used"`
		URL     string `json:"url,omitempty"`
	} `json:"simulation"`
}