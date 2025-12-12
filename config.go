package p2p

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	msgbus "github.com/bsv-blockchain/go-p2p-message-bus"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/viper"
)

var defaultBootstrapPeers = map[string][]string{
	"main": {
		"/dns4/teranode-eks-mainnet-us-1-p2p.bsvb.tech/tcp/9905/p2p/12D3KooWH5JVqGdaw7JEizmysCfRRcPGTFfvRJF7Hkure7oQWYnb",
		"/dns4/teranode-eks-mainnet-eu-1-p2p.bsvb.tech/tcp/9905/p2p/12D3KooW9z2JRV37TqsmU8sDQcSQDZGSgtPpvWUmVegYxYvXfW9H",
	},
	"test": {
		"/dns4/teranode-eks-testnet-us-1-p2p.bsvb.tech/tcp/9905/p2p/12D3KooWK7tQiJHKp4TmS632XTXy7nScvVvyL7Qx5YiU65EYnRub",
		"/dns4/teranode-eks-testnet-eu-2-p2p.bsvb.tech/tcp/9905/p2p/12D3KooWR9DMm622shDLAe5hQZk4phNERF84S77JocXfLyZU9NsF",
	},
	"stn": {
		"/dns4/teranode-eks-ttn-us-1-p2p.bsvb.tech/tcp/9905/p2p/12D3KooWFj5nh1m3iAooxnfp5VvDtufYajTpBSopUt7anj4XLqJp",
		"/dns4/teranode-eks-ttn-eu-1-p2p.bsvb.tech/tcp/9905/p2p/12D3KooWDnQoDerA2KC8xD5hDqiSp21zf9zS5ezM32wuXgLUaden",
	},
}

// LoadOrGeneratePrivateKey loads a P2P private key from the given storage path,
// or generates a new one if it doesn't exist.
func LoadOrGeneratePrivateKey(storagePath string) (crypto.PrivKey, error) {
	keyPath := filepath.Join(storagePath, "p2p_key.hex")

	if data, err := os.ReadFile(keyPath); err == nil {
		privKey, err := msgbus.PrivateKeyFromHex(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key from %s: %w", keyPath, err)
		}
		return privKey, nil
	}

	privKey, err := msgbus.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	keyHex, err := msgbus.PrivateKeyToHex(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode private key: %w", err)
	}

	if err := os.MkdirAll(storagePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(keyHex), 0o600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	return privKey, nil
}

func (c *Config) SetDefaults(v *viper.Viper, configPath string) {
	prefix := ""
	if configPath != "" {
		prefix = configPath + "."
	}

	v.SetDefault(prefix+"network", "main")
	v.SetDefault(prefix+"msgbus.dht_mode", "off")
	v.SetDefault(prefix+"msgbus.max_connections", 35)
	v.SetDefault(prefix+"msgbus.min_connections", 25)
	v.SetDefault(prefix+"msgbus.connection_grace_period", 20*time.Second)
	v.SetDefault(prefix+"msgbus.peer_cache_ttl", 24*time.Hour)
}

// Initialize applies defaults for zero-value fields and creates a new Client.
// Name is required and identifies this client on the P2P network.
func (c *Config) Initialize(_ context.Context, name string) (*Client, error) {
	// Apply defaults for zero-value fields
	if c.StoragePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			c.StoragePath = "./.teranode-p2p"
		} else {
			c.StoragePath = path.Join(homeDir, ".teranode-p2p")
		}
	} else if len(c.StoragePath) >= 2 && c.StoragePath[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve home directory for storage path: %w", err)
		}
		c.StoragePath = path.Join(homeDir, c.StoragePath[2:])
	}

	// Apply dependent defaults
	if len(c.MsgBus.BootstrapPeers) == 0 {
		c.MsgBus.BootstrapPeers = defaultBootstrapPeers[c.Network]
	}
	if c.MsgBus.PeerCacheFile == "" {
		c.MsgBus.PeerCacheFile = filepath.Join(c.StoragePath, "peer_cache.json")
	}

	// Load or generate private key if not provided
	if c.MsgBus.PrivateKey == nil {
		privKey, err := LoadOrGeneratePrivateKey(c.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load/generate private key: %w", err)
		}
		c.MsgBus.PrivateKey = privKey
	}

	// Apply msgbus defaults
	c.MsgBus.Name = name

	return NewClient(*c)
}
