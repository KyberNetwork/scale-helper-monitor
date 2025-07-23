package monitor

import (
	"math/big"
	"time"

	distributor "scale-helper-monitor/internal/config/distributor"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// RootSubmittedEvent represents the parsed event data
type RootSubmittedEvent struct {
	CampaignId         [32]byte
	PendingRoot        [32]byte
	EffectiveTimestamp *big.Int
	BlockNumber        uint64
	TxHash             string
	Timestamp          time.Time
	Network            *distributor.Network
}

// Monitor handles the event monitoring logic for a single network
type Monitor struct {
	Client       *ethclient.Client
	SlackClient  *slack.Client
	Config       *distributor.Config
	Network      *distributor.Network
	Logger       *logrus.Logger
	ContractABI  abi.ABI
	StateManager *StateManager
}

// MultiNetworkMonitor manages multiple network monitors
type MultiNetworkMonitor struct {
	Monitors []*Monitor
	Config   *distributor.Config
	Logger   *logrus.Logger
}
