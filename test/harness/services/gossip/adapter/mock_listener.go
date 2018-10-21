package adapter

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type mockListener struct {
	mock.Mock
}

func (m *mockListener) OnTransportMessageReceived(ctx context.Context, payloads [][]byte) {
	m.Called(ctx, payloads)
}

func listenTo(transport adapter.Transport, publicKey primitives.Ed25519PublicKey) *mockListener {
	l := &mockListener{}
	transport.RegisterListener(l, publicKey)
	return l
}

func (m *mockListener) expectReceive(payloads [][]byte) {
	m.WhenOnTransportMessageReceived(payloads).Return().Times(1)
}

func (m *mockListener) expectNotReceive() {
	m.Never("OnTransportMessageReceived", mock.Any, mock.Any)
}

func (m *mockListener) WhenOnTransportMessageReceived(arg interface{}) *mock.MockFunction {
	return m.When("OnTransportMessageReceived", mock.Any, arg)
}
