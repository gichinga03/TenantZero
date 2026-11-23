// Package wsclient wraps a gorilla/websocket connection to the TenantZero server.
package wsclient

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gorilla/websocket"
)

// outMsg is the wire format for messages sent to the server.
type outMsg struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// inMsg is the wire format for messages received from the server.
type inMsg struct {
	Type      string `json:"type"`
	EventType string `json:"event_type,omitempty"`
	Payload   string `json:"payload"`
}

// Client manages a single WebSocket connection to the TenantZero server.
type Client struct {
	conn      *websocket.Conn
	token     string
	serverURL string
}

// New constructs a Client. Call Connect before Send/Recv.
func New(serverURL, token string) *Client {
	return &Client{serverURL: serverURL, token: token}
}

// Connect dials the server WebSocket endpoint at /ws?token=<jwt>.
func (c *Client) Connect() error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("wsclient: parse server URL: %w", err)
	}
	// Convert http(s) scheme to ws(s) if the caller passed HTTP URLs.
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}
	u.Path = "/ws"
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("wsclient: dial %s: %w", u.String(), err)
	}
	c.conn = conn
	return nil
}

// Send serialises input as a command message and writes it to the server.
func (c *Client) Send(input string) error {
	msg := outMsg{Type: "command", Payload: input}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("wsclient: marshal: %w", err)
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("wsclient: write: %w", err)
	}
	return nil
}

// Recv reads one message from the server and returns its parsed fields.
// Blocks until a message arrives or the connection closes.
func (c *Client) Recv() (msgType, eventType, payload string, err error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return "", "", "", fmt.Errorf("wsclient: read: %w", err)
	}
	var msg inMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", "", "", fmt.Errorf("wsclient: unmarshal: %w", err)
	}
	return msg.Type, msg.EventType, msg.Payload, nil
}

// Close closes the underlying WebSocket connection.
func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
	}
}