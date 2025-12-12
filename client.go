package p2p

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	msgbus "github.com/bsv-blockchain/go-p2p-message-bus"
	teranode "github.com/bsv-blockchain/teranode/services/p2p"
)

// Client provides a high-level interface for subscribing to Teranode P2P messages.
// It supports multiple subscribers per topic via internal fan-out.
type Client struct {
	msgbus  msgbus.Client
	logger  *slog.Logger
	network string // P2P network name (e.g., "mainnet", "testnet")

	// Fan-out support: multiple subscribers per topic
	mu           sync.RWMutex
	blockSubs    []chan teranode.BlockMessage
	subtreeSubs  []chan teranode.SubtreeMessage
	rejectedSubs []chan teranode.RejectedTxMessage
	statusSubs   []chan teranode.NodeStatusMessage

	// Track if we've started listening to each topic
	blockStarted    bool
	subtreeStarted  bool
	rejectedStarted bool
	statusStarted   bool
}

// NewClient creates a new Teranode P2P client.
// Prefer using Config.Initialize() which applies defaults before creating the client.
func NewClient(cfg Config) (*Client, error) {
	p2pClient, err := msgbus.NewClient(cfg.MsgBus)
	if err != nil {
		return nil, err
	}

	return &Client{
		msgbus:  p2pClient,
		logger:  slog.Default(),
		network: cfg.Network,
	}, nil
}

// GetID returns this client's peer ID.
func (c *Client) GetID() string {
	return c.msgbus.GetID()
}

// GetNetwork returns the network this client is connected to.
func (c *Client) GetNetwork() string {
	return c.network
}

// Close shuts down the P2P client and closes all subscriber channels.
func (c *Client) Close() error {
	c.mu.Lock()
	for _, ch := range c.blockSubs {
		close(ch)
	}
	c.blockSubs = nil
	for _, ch := range c.subtreeSubs {
		close(ch)
	}
	c.subtreeSubs = nil
	for _, ch := range c.rejectedSubs {
		close(ch)
	}
	c.rejectedSubs = nil
	for _, ch := range c.statusSubs {
		close(ch)
	}
	c.statusSubs = nil
	c.mu.Unlock()

	return c.msgbus.Close()
}

// GetPeers returns information about all known peers.
func (c *Client) GetPeers() []msgbus.PeerInfo {
	return c.msgbus.GetPeers()
}

// SubscribeBlocks subscribes to block announcements.
// Multiple callers can subscribe; each receives all messages (fan-out).
// The returned channel is closed when the client is closed or context is cancelled.
func (c *Client) SubscribeBlocks(ctx context.Context) <-chan teranode.BlockMessage {
	out := make(chan teranode.BlockMessage, 100)

	c.mu.Lock()
	c.blockSubs = append(c.blockSubs, out)

	// Start the topic listener only once
	if !c.blockStarted {
		c.blockStarted = true
		topic := TopicName(c.network, TopicBlock)
		rawChan := c.msgbus.Subscribe(topic)
		go c.fanOutBlocks(rawChan, topic)
	}
	c.mu.Unlock()

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		c.mu.Lock()
		for i, ch := range c.blockSubs {
			if ch == out {
				c.blockSubs = append(c.blockSubs[:i], c.blockSubs[i+1:]...)
				close(out)
				break
			}
		}
		c.mu.Unlock()
	}()

	return out
}

// SubscribeSubtrees subscribes to subtree (transaction batch) announcements.
// Multiple callers can subscribe; each receives all messages (fan-out).
func (c *Client) SubscribeSubtrees(ctx context.Context) <-chan teranode.SubtreeMessage {
	out := make(chan teranode.SubtreeMessage, 100)

	c.mu.Lock()
	c.subtreeSubs = append(c.subtreeSubs, out)

	if !c.subtreeStarted {
		c.subtreeStarted = true
		topic := TopicName(c.network, TopicSubtree)
		rawChan := c.msgbus.Subscribe(topic)
		go c.fanOutSubtrees(rawChan, topic)
	}
	c.mu.Unlock()

	go func() {
		<-ctx.Done()
		c.mu.Lock()
		for i, ch := range c.subtreeSubs {
			if ch == out {
				c.subtreeSubs = append(c.subtreeSubs[:i], c.subtreeSubs[i+1:]...)
				close(out)
				break
			}
		}
		c.mu.Unlock()
	}()

	return out
}

// SubscribeRejectedTxs subscribes to rejected transaction notifications.
// Multiple callers can subscribe; each receives all messages (fan-out).
func (c *Client) SubscribeRejectedTxs(ctx context.Context) <-chan teranode.RejectedTxMessage {
	out := make(chan teranode.RejectedTxMessage, 100)

	c.mu.Lock()
	c.rejectedSubs = append(c.rejectedSubs, out)

	if !c.rejectedStarted {
		c.rejectedStarted = true
		topic := TopicName(c.network, TopicRejectedTx)
		rawChan := c.msgbus.Subscribe(topic)
		go c.fanOutRejectedTxs(rawChan, topic)
	}
	c.mu.Unlock()

	go func() {
		<-ctx.Done()
		c.mu.Lock()
		for i, ch := range c.rejectedSubs {
			if ch == out {
				c.rejectedSubs = append(c.rejectedSubs[:i], c.rejectedSubs[i+1:]...)
				close(out)
				break
			}
		}
		c.mu.Unlock()
	}()

	return out
}

// SubscribeNodeStatus subscribes to node status updates.
// Multiple callers can subscribe; each receives all messages (fan-out).
func (c *Client) SubscribeNodeStatus(ctx context.Context) <-chan teranode.NodeStatusMessage {
	out := make(chan teranode.NodeStatusMessage, 100)

	c.mu.Lock()
	c.statusSubs = append(c.statusSubs, out)

	if !c.statusStarted {
		c.statusStarted = true
		topic := TopicName(c.network, TopicNodeStatus)
		rawChan := c.msgbus.Subscribe(topic)
		go c.fanOutNodeStatus(rawChan, topic)
	}
	c.mu.Unlock()

	go func() {
		<-ctx.Done()
		c.mu.Lock()
		for i, ch := range c.statusSubs {
			if ch == out {
				c.statusSubs = append(c.statusSubs[:i], c.statusSubs[i+1:]...)
				close(out)
				break
			}
		}
		c.mu.Unlock()
	}()

	return out
}

// Fan-out goroutines - one per topic type

func (c *Client) fanOutBlocks(rawChan <-chan msgbus.Message, topic string) {
	for msg := range rawChan {
		var typed teranode.BlockMessage
		if err := json.Unmarshal(msg.Data, &typed); err != nil {
			c.logger.Error("failed to unmarshal block message",
				slog.String("topic", topic),
				slog.String("error", err.Error()))
			continue
		}

		c.mu.RLock()
		subs := make([]chan teranode.BlockMessage, len(c.blockSubs))
		copy(subs, c.blockSubs)
		c.mu.RUnlock()

		for _, ch := range subs {
			select {
			case ch <- typed:
			default:
				// Subscriber is slow, skip to avoid blocking
			}
		}
	}
}

func (c *Client) fanOutSubtrees(rawChan <-chan msgbus.Message, topic string) {
	for msg := range rawChan {
		var typed teranode.SubtreeMessage
		if err := json.Unmarshal(msg.Data, &typed); err != nil {
			c.logger.Error("failed to unmarshal subtree message",
				slog.String("topic", topic),
				slog.String("error", err.Error()))
			continue
		}

		c.mu.RLock()
		subs := make([]chan teranode.SubtreeMessage, len(c.subtreeSubs))
		copy(subs, c.subtreeSubs)
		c.mu.RUnlock()

		for _, ch := range subs {
			select {
			case ch <- typed:
			default:
			}
		}
	}
}

func (c *Client) fanOutRejectedTxs(rawChan <-chan msgbus.Message, topic string) {
	for msg := range rawChan {
		var typed teranode.RejectedTxMessage
		if err := json.Unmarshal(msg.Data, &typed); err != nil {
			c.logger.Error("failed to unmarshal rejected-tx message",
				slog.String("topic", topic),
				slog.String("error", err.Error()))
			continue
		}

		c.mu.RLock()
		subs := make([]chan teranode.RejectedTxMessage, len(c.rejectedSubs))
		copy(subs, c.rejectedSubs)
		c.mu.RUnlock()

		for _, ch := range subs {
			select {
			case ch <- typed:
			default:
			}
		}
	}
}

func (c *Client) fanOutNodeStatus(rawChan <-chan msgbus.Message, topic string) {
	for msg := range rawChan {
		var typed teranode.NodeStatusMessage
		if err := json.Unmarshal(msg.Data, &typed); err != nil {
			c.logger.Error("failed to unmarshal node_status message",
				slog.String("topic", topic),
				slog.String("error", err.Error()))
			continue
		}

		c.mu.RLock()
		subs := make([]chan teranode.NodeStatusMessage, len(c.statusSubs))
		copy(subs, c.statusSubs)
		c.mu.RUnlock()

		for _, ch := range subs {
			select {
			case ch <- typed:
			default:
			}
		}
	}
}
