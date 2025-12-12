package p2p

import (
	msgbus "github.com/bsv-blockchain/go-p2p-message-bus"
)

// Config holds configuration for creating a new Client.
// It embeds msgbus.Config to expose all underlying P2P options.
// Use mapstructure tags for viper compatibility when embedding in parent configs.
type Config struct {
	// Network is the Bitcoin network to connect to (e.g., "main", "test", "stn")
	Network string `mapstructure:"network"`

	// StoragePath is the directory for storing persistent data (p2p_key.hex, peer_cache.json).
	StoragePath string `mapstructure:"storage_path"`

	// Embed the full P2P message bus config for all underlying options.
	// Fields like Name, Port, BootstrapPeers, DHTMode, MaxConnections, Logger, etc.
	// are all available. Use squash to flatten in viper/mapstructure.
	MsgBus msgbus.Config `mapstructure:"msgbus"`
}
