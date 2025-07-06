package monitor

import "fmt"

// CallGetScaledInputDataError represents an error from callGetScaledInputData
type CallGetScaledInputDataError struct {
	ChainName string `json:"chain_name"`
	Message   string `json:"message"`
	Data      string `json:"data,omitempty"`
}

func (e *CallGetScaledInputDataError) Error() string {
	return fmt.Sprintf("Scale Helper Error: %s", e.Message)
}
