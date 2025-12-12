# go-teranode-p2p-client

Go library for connecting to Teranode's P2P gossip network as a client/subscriber.

## Features

- Canonical message types matching Teranode's definitions
- Embedded bootstrap peers for mainnet, testnet, and STN
- Persistent P2P identity (key management)
- Topic name helpers

## Installation

```bash
go get github.com/bsv-blockchain/go-teranode-p2p-client
```

## Usage

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    p2pclient "github.com/bsv-blockchain/go-teranode-p2p-client"
    p2p "github.com/bsv-blockchain/go-p2p-message-bus"
)

func main() {
    ctx := context.Background()

    // Load or generate a persistent P2P identity
    privKey, err := p2pclient.LoadOrGeneratePrivateKey("./data")
    if err != nil {
        log.Fatal(err)
    }

    // Create P2P client with Teranode bootstrap peers
    client, err := p2p.NewClient(p2p.Config{
        Name:           "my-app",
        PrivateKey:     privKey,
        BootstrapPeers: p2pclient.BootstrapPeers("main"),
        PeerCacheFile:  "./data/peer_cache.json",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Subscribe to block announcements
    blockTopic := p2pclient.TopicName(p2pclient.NetworkMainnet, p2pclient.TopicBlock)
    blockChan := client.Subscribe(blockTopic)

    for msg := range blockChan {
        var blockMsg p2pclient.BlockMessage
        if err := json.Unmarshal(msg.Data, &blockMsg); err != nil {
            log.Printf("Error parsing block message: %v", err)
            continue
        }
        log.Printf("New block: height=%d hash=%s", blockMsg.Height, blockMsg.Hash)
    }
}
```

## Message Types

- `BlockMessage` - New block announcements
- `SubtreeMessage` - Transaction batch (subtree) announcements
- `RejectedTxMessage` - Rejected transaction notifications
- `NodeStatusMessage` - Node status updates

## Topics

| Constant | Topic Name |
|----------|------------|
| `TopicBlock` | `block` |
| `TopicSubtree` | `subtree` |
| `TopicRejectedTx` | `rejected-tx` |
| `TopicNodeStatus` | `node_status` |

Use `TopicName(network, topic)` to construct full topic names (e.g., `teranode/bitcoin/1.0.0/mainnet-block`).

## Networks

| Constant | Network |
|----------|---------|
| `NetworkMainnet` | `mainnet` |
| `NetworkTestnet` | `testnet` |
| `NetworkSTN` | `stn` |
| `NetworkTeratestnet` | `teratestnet` |

## Bootstrap Peers

Embedded bootstrap peers are available for `main`, `test`, and `stn` networks via `BootstrapPeers(network)`.
