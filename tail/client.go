package tail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type TailLogClient struct {
	AccountID  string
	APIToken   string
	WorkerName string
	Env *string

	expiryAt time.Time

	mu   sync.Mutex
	conn *websocket.Conn

	wg sync.WaitGroup
}

// TailResponse represents the Cloudflare API response schema.
type TailResponse struct {
	Result struct {
		ID        string    `json:"id"`
		URL       string    `json:"url"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"result"`
	Success bool `json:"success"`
}

// NewTailLogClient initializes a client with default settings.
func NewTailLogClient(accountID, apiToken, workerName string, env *string) *TailLogClient {
	return &TailLogClient{
		AccountID:  accountID,
		WorkerName: workerName,
		APIToken:   apiToken,
		Env: env,
		expiryAt: time.Time{},

		mu: sync.Mutex{},
		conn: nil,
		wg: sync.WaitGroup{},
	}
}

func (c *TailLogClient) getTailURI() string {
	if c.Env != nil {
		return fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s/%s/tails", c.AccountID, c.WorkerName, *c.Env)
	}
	return fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s/tails", c.AccountID, c.WorkerName)
}

func (c *TailLogClient) CreateSession(ctx context.Context) (*TailResponse, error) {
	apiURL := c.getTailURI()
	
	payload := map[string]any{"filters": []string{}}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare api error: status %d", resp.StatusCode)
	}

	var res TailResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	c.expiryAt = res.Result.ExpiresAt
	return &res, nil
}

func (c *TailLogClient) StreamLogs(ctx context.Context, wsURL string) (chan []byte, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	headers.Set("Sec-WebSocket-Protocol", "trace-v1")

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	log.Printf("Connected to stream for worker: %s", c.WorkerName)

	logChan := make(chan []byte, 100)

	c.wg.Add(1)

	go func() {
		defer close(logChan)
		defer c.wg.Done()
		
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("Websocket read error: %v", err)
				}
				return
			}

			logChan <- message
		}
	}()

	return logChan, nil
}

func (c *TailLogClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 2. Close the connection
	if c.conn != nil {
		_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	c.wg.Wait()

	return nil
}