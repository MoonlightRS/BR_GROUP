package ascendex

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testBBOChannel = "api/marketDataStream:mock"
)

func TestConnection(t *testing.T) {
	c := NewAPIClient()
	err := c.Connection()
	assert.NoError(t, err)
	err = c.Disconnect()
	assert.NoError(t, err)
}

func TestSubscribeToChannel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	c := NewAPIClient()
	err := c.Connection()
	assert.NoError(t, err)

	if err := c.SubscribeToChannel(ctx, testBBOChannel); err != nil {
		t.Error(err)
	}

	select {
	case <-ctx.Done():
		t.Error("timeout while waiting for message")
	case msg := <-c.ReadMessagesFromChannel():
		assert.NotEmpty(t, msg.Symbol)
		assert.NotEqual(t, float64(0), msg.Bid)
		assert.NotEqual(t, float64(0), msg.Ask)
	}

	err = c.Disconnect()
	assert.NoError(t, err)
}

func TestWriteMessagesToChannel(t *testing.T) {
	c := NewAPIClient()
	err := c.Connection()
	assert.NoError(t, err)
	c.lastSentHeartbeat = time.Time{}
	c.sendHeartbeat()
	assert.NotEqual(t, time.Time{}, c.lastSentHeartbeat)
	err = c.Disconnect()
	assert.NoError(t, err)
}

func TestConnection(t *testing.T) {
	c := NewAPIClient()
	err := c.Connection()
	assert.NoError(t, err)
	err = c.Disconnect()
	assert.NoError(t, err)
}

func TestWriteMessagesToChannel(t *testing.T) {
	c := NewAPIClient()
	err := c.Connection()
	assert.NoError(t, err)
	c.lastSentHeartbeat = time.Time{}
	c.sendHeartbeat()
	assert.NotEqual(t, time.Time{}, c.lastSentHeartbeat)
	err = c.Disconnect()
	assert.NoError(t, err)
}

func TestDisconnect(t *testing.T) {
	c := NewAPIClient()
	err := c.Connection()
	assert.NoError(t, err)
	err = c.Disconnect()
	assert.NoError(t, err)
	err = c.Disconnect()
	assert.NoError(t, err)
}
