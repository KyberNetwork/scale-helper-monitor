package monitor

import "fmt"

// CallGetScaledInputDataError represents an error from callGetScaledInputData
type CallGetScaledInputDataError struct {
	ChainID int    `json:"chain_id"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *CallGetScaledInputDataError) Error() string {
	return fmt.Sprintf("CallGetScaledInputData Error [Chain %d]: %s", e.ChainID, e.Message)
} 