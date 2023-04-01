package ascendex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	baseURL = "wss://ascendex.com/0/api/pro/v1/stream"
)

type APIClient struct {
	conn              *websocket.Conn
	URL               *url.URL
	done              chan struct{}
	errorHandler      func(error)
	lastSentHeartbeat interface{}
}

type message struct {
	Type string `json:"type"`
	Data data   `json:"data"`
}

type data struct {
	Symbol    string  `json:"s"`
	BidPrice  float64 `json:"bp"`
	BidAmount float64 `json:"ba"`
	AskPrice  float64 `json:"ap"`
	AskAmount float64 `json:"aa"`
}

func NewAPIClient(errorHandler func(error)) (*APIClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &APIClient{
		URL:          u,
		done:         make(chan struct{}),
		errorHandler: errorHandler,
	}, nil
}

func (c *APIClient) Connection() error {
	header := http.Header{}
	header.Add("Connection", "Upgrade")
	header.Add("Upgrade", "websocket")
	header.Add("Sec-WebSocket-Version", "13")
	header.Add("Sec-WebSocket-Protocol", "json")
	header.Add("Sec-WebSocket-Extensions", "permessage-deflate; client_max_window_bits")

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: time.Second * 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := dialer.DialContext(ctx, c.URL.String(), header)
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusSwitchingProtocols {
			return fmt.Errorf("websocket connect error status %d: %w",
				resp.StatusCode, err)
		}
		return fmt.Errorf("websocket connect error: %w", err)
	}

	c.conn = conn

	go c.loop()
	return nil
}

func (c *APIClient) Disconnect() {
	close(c.done)
	if err := c.conn.Close(); err != nil {
		log.Printf("error: %+v\n", err)
	}
}

func (c *APIClient) SubscribeToChannel(symbol string, ch chan<- BBO) error {
	if c.conn == nil {
		return errors.New("websocket not connected")
	}

	message := map[string]interface{}{
		"action": "subscribe",
		"ch":     fmt.Sprintf("orderbook:%s", symbol),
	}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe message: %w", err)
	}

	err = c.conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	go c.ReadMessagesFromChannel(ch)

	return nil
}

func (c *APIClient) ReadMessagesFromChannel(ch chan<- BBO) {
	for {
		select {
		case <-c.done:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				c.errorHandler(err)
				continue
			}

			var (
				msg message
			)
			err = json.Unmarshal(message, &msg)
			if err != nil {
				c.errorHandler(err)
				continue
			}

			if msg.Type != "snapshot" {
				continue
			}

			data := msg.Data
			bbo := BBO{
				Symbol:    data.Symbol,
				BidPrice:  data.BidPrice,
				BidAmount: data.BidAmount,
				AskPrice:  data.AskPrice,
				AskAmount: data.AskAmount,
			}

			ch <- bbo
		}
	}
}

func (c *APIClient) WriteMessagesToChannel(done chan struct{}) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			err := c.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				c.errorHandler(err)
			}
		}
	}
}

func (c *APIClient) loop() {
	doneChan := make(chan struct{})

	go c.WriteMessagesToChannel(doneChan)

	for {
		select {
		case <-c.done:
			close(doneChan)
			return
		}
	}
}
