package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const (
	maxMessageSize = 512
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
)

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client ...
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	uid    string
	name   string
	buffer chan []byte
}

// Message ...
type Message struct {
	Source     string      `json:"source"`
	FromUID    string      `json:"from_uid,omitempty"`
	FromName   string      `json:"from_name,omitempty"`
	ToUID      string      `json:"to_uid,omitempty"`
	ReceivedAt time.Time   `json:"received_at"`
	Data       interface{} `json:"data"`
}

// IsAvailableFor ...
func (message *Message) IsAvailableFor(client *Client) bool {
	return message.FromUID != "" && message.FromUID == client.uid || message.ToUID != "" && message.ToUID != client.uid
}

// Hub ...
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

// NewHub ...
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func formatMessage(message *Message) ([]byte, error) {
	// []byte(fmt.Sprintf(`{"fromName": "%v", "fromUID": %v, "data": "%v"}`, message.fromName, message.fromUID, message.data))
	return json.Marshal(message)
}

// Run ...
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			log.Println("Registered client uid:", client.uid)
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.buffer)
				log.Println("Unregistered client uid:", client.uid)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				if message.IsAvailableFor(client) {
					continue
				}
				data, err := formatMessage(message)
				if err != nil {
					continue
				}
				select {
				case client.buffer <- data:
				default:
					close(client.buffer)
					delete(h.clients, client)
				}
			}
		}
	}
}

// Authorize ...
func (h *Hub) Authorize(token string) (string, string, error) {
	var uid string
	var name string
	var err error
	uid = fmt.Sprintf("%v", rand.Intn(100000000))
	name = "test"
	return uid, name, err
}

// WriteError ...
func (c *Client) WriteError(message string) {
	messageData := make(map[string]interface{})
	messageData["error"] = message
	c.hub.broadcast <- &Message{
		Source: "server",
		ToUID:  c.uid,
		Data:   messageData,
	}
	c.hub.unregister <- c
}

// Reader ...
func (c *Client) Reader() {
	defer func() {
		// c.hub.unregister <- c
		// c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			errors.WithStack(err)
			c.WriteError("Internal Error")
			break
		}
		message := string(bytes.TrimSpace(bytes.Replace(data, []byte{'\n'}, []byte{' '}, -1)))
		if len(message) < 1 {
			c.WriteError("Invalid data")
			break
		}
		messageData := new(map[string]interface{})
		if err := json.Unmarshal([]byte(message), messageData); err != nil {
			c.WriteError("Invalid data")
			break
		}
		c.hub.broadcast <- &Message{
			Source:   "client",
			FromUID:  c.uid,
			FromName: c.name,
			Data:     messageData,
		}
	}
}

// Writer ...
func (c *Client) Writer() {
	pingTicker := time.NewTicker(pingPeriod)

	defer func() {
		pingTicker.Stop()
	}()

	for {
		select {
		case message, ok := <-c.buffer:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write([]byte("["))
			w.Write(message)
			for i := 0; i < len(c.buffer); i++ {
				w.Write([]byte(","))
				w.Write(<-c.buffer)
			}
			w.Write([]byte("]"))
			if err := w.Close(); err != nil {
				return
			}
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// EchoHandler ...
func EchoHandler(hub *Hub, c buffalo.Context) error {
	log.Println("Handling Websocket connection from", c.Request().RemoteAddr)

	uid, name, err := hub.Authorize(c.Request().URL.Query().Get("token"))

	if err != nil {
		c.Response().WriteHeader(403)
		log.Println(err)
		errors.WithStack(err)
	}

	ws, err := websocket.Upgrade(c.Response(), c.Request(), c.Response().Header(), 1024, 1024)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			errors.WithStack(err)
		}
	}

	client := &Client{hub: hub, conn: ws, uid: uid, name: name, buffer: make(chan []byte, 256)}
	client.hub.register <- client

	go client.Reader()
	client.Writer()

	client.hub.unregister <- client
	client.conn.Close()

	return nil
}
