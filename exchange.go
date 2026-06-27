package umi

import (
	"fmt"
	"net"
	"sync"
)

// ============================================================================
// 1. EXCHANGE LAYER (交换层类型定义)
// ============================================================================

type MessageType string

const (
	TypeQuery  MessageType = "QUERY"
	TypeStream MessageType = "STREAM"
	TypeEvent  MessageType = "EVENT"
)

// Envelope 消息信封
type Envelope struct {
	Type    MessageType
	Source  string
	Target  string
	Payload interface{}
}

// QueryPayload 用于 REQ-REP 语义
type QueryPayload struct {
	Question string
	ReplyCh  chan string
}

// StreamPayload 用于直连双向流 (Sans-IO / Proxy 场景核心)
type StreamPayload struct {
	Conn net.Conn
}

// ExchangeBus 负责单机实例间的信箱分发
type ExchangeBus struct {
	mu        sync.RWMutex
	mailboxes map[string]chan Envelope
}

func NewExchangeBus() *ExchangeBus {
	return &ExchangeBus{
		mailboxes: make(map[string]chan Envelope),
	}
}

func (b *ExchangeBus) Register(instanceID string, ch chan Envelope) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mailboxes[instanceID] = ch
}

func (b *ExchangeBus) Unregister(instanceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.mailboxes, instanceID)
}

func (b *ExchangeBus) Send(env Envelope) error {
	b.mu.RLock()
	ch, exists := b.mailboxes[env.Target]
	b.mu.RUnlock()

	if !exists {
		return fmt.Errorf("target instance [%s] not found", env.Target)
	}

	// 异步非阻塞投递，防止单个实例卡死导致总线崩溃
	select {
	case ch <- env:
		return nil
	default:
		return fmt.Errorf("target instance [%s] mailbox full", env.Target)
	}
}
