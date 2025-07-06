package kyberswap

// KyberSwapRoute represents a route response from KyberSwap API
type KyberSwapRoute struct {
	TokenIn       string            `json:"tokenIn"`
	AmountIn      string            `json:"amountIn"`
	AmountInUsd   string            `json:"amountInUsd"`
	TokenOut      string            `json:"tokenOut"`
	AmountOut     string            `json:"amountOut"`
	AmountOutUsd  string            `json:"amountOutUsd"`
	Gas           string            `json:"gas"`
	GasPrice      string            `json:"gasPrice"`
	GasUSD        string            `json:"gasUsd"`
	ExtraFee      *ExtraFee         `json:"extraFee,omitempty"`
	Route         [][]KyberSwapSwap `json:"route"`
	RouteID       string            `json:"routeID"`
	Checksum      string            `json:"checksum"`
	Timestamp     int64             `json:"timestamp"`
	RouterAddress string            `json:"routerAddress,omitempty"`
}

// KyberSwapRouteEncodedData represents encoded route data
type KyberSwapRouteEncodedData struct {
	AmountIn              string `json:"amountIn"`
	AmountInUsd           string `json:"amountInUsd"`
	AmountOut             string `json:"amountOut"`
	AmountOutUsd          string `json:"amountOutUsd"`
	Gas                   string `json:"gas"`
	GasUSD                string `json:"gasUsd"`
	AdditionalCostUsd     string `json:"additionalCostUsd"`
	AdditionalCostMessage string `json:"additionalCostMessage"`
	Data                  string `json:"data"`
	RouterAddress         string `json:"routerAddress"`
	TransactionValue      string `json:"transactionValue"`
}

// KyberSwapAPIResponse represents the API response structure
type KyberSwapAPIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		RouteSummary  KyberSwapRoute `json:"routeSummary"`
		RouterAddress string         `json:"routerAddress"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

// KyberSwapEncodedData represents encoded data response
type KyberSwapEncodedData struct {
	Code      int                       `json:"code"`
	Message   string                    `json:"message"`
	Data      KyberSwapRouteEncodedData `json:"data"`
	RequestID string                    `json:"requestId"`
}

// ExtraFee represents fee information in the route
type ExtraFee struct {
	FeeAmount   string `json:"feeAmount"`
	ChargeFeeBy string `json:"chargeFeeBy"`
	IsInBps     bool   `json:"isInBps"`
	FeeReceiver string `json:"feeReceiver"`
}

// KyberSwapSwap represents a single swap in the route
type KyberSwapSwap struct {
	Pool       string                 `json:"pool"`
	TokenIn    string                 `json:"tokenIn"`
	TokenOut   string                 `json:"tokenOut"`
	SwapAmount string                 `json:"swapAmount"`
	AmountOut  string                 `json:"amountOut"`
	Exchange   string                 `json:"exchange"`
	PoolType   string                 `json:"poolType"`
	PoolExtra  map[string]interface{} `json:"poolExtra,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}
