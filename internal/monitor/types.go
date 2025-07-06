package monitor

import (
	"scale-helper-monitor/internal/clients/kyberswap"
)

// Config represents the monitoring configuration
type Config struct {
	Interval string `mapstructure:"interval"`
	Timeout  string `mapstructure:"timeout"`
}

// ChainConfig represents blockchain configuration
type ChainConfig struct {
	Name            string `mapstructure:"name"`
	ChainID         int    `mapstructure:"chain_id"`
	RPCURL          string `mapstructure:"rpc_url"`
	ContractAddress string `mapstructure:"contract_address"`
}

// TokenInfo represents token information including slot, symbol, amount, and decimal
type TokenInfo struct {
	Slot     string `json:"slot"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

type TestCase struct {
	ChainName string `mapstructure:"chain_name"`
	TokenIn   string `mapstructure:"token_in"`
	TokenOut  string `mapstructure:"token_out"`
	Amount    string `mapstructure:"amount"`
}

// Result represents the result of a monitoring check
type Result struct {
	ChainName           string                      `json:"chain_name"`
	TokenIn             string                      `json:"token_in"`
	TokenOut            string                      `json:"token_out"`
	Amount              string                      `json:"amount"`
	IsSuccess           bool                        `json:"is_success"`
	ReturnedData        string                      `json:"returned_data"`
	InputData           string                      `json:"input_data"`
	Route               [][]kyberswap.KyberSwapSwap `json:"route"`
	NewAmount           string                      `json:"new_amount"`
	Error               string                      `json:"error,omitempty"`
	OriginalTenderlyURL string                      `json:"original_tenderly_url,omitempty"`
	ScaledTenderlyURL   string                      `json:"scaled_tenderly_url,omitempty"`
}

// ContractCallResult represents the result of calling getScaledInputData
type ContractCallResult struct {
	IsSuccess bool
	Data      []byte
}

// Implement the slack.MonitoringResult interface
func (r *Result) GetChainName() string                  { return r.ChainName }
func (r *Result) GetTokenIn() string                    { return r.TokenIn }
func (r *Result) GetTokenOut() string                   { return r.TokenOut }
func (r *Result) GetAmount() string                     { return r.Amount }
func (r *Result) GetIsSuccess() bool                    { return r.IsSuccess }
func (r *Result) GetError() string                      { return r.Error }
func (r *Result) GetInputData() string                  { return r.InputData }
func (r *Result) GetReturnedData() string               { return r.ReturnedData }
func (r *Result) GetRoute() [][]kyberswap.KyberSwapSwap { return r.Route }
func (r *Result) GetNewAmount() string                  { return r.NewAmount }
func (r *Result) GetOriginalTenderlyURL() string        { return r.OriginalTenderlyURL }
func (r *Result) GetScaledTenderlyURL() string          { return r.ScaledTenderlyURL }
