/*
   Copyright (c) 2022-present, Adil Alper DALKIRAN

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package signaling

import (
	"bytes"
	"encoding/json"
	"time"

	logging "github.com/adalkiran/go-colorful-logging"

	"github.com/gorilla/websocket"
)

// See: https://github.com/gorilla/websocket/tree/master/examples/chat

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 81920
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type WsClient struct {
	id int

	wsHub *WsHub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	SendMessage chan WsMessageContainer
}

type ReceivedMessage struct {
	Sender  *WsClient
	Message []byte
}

type ClientWelcomeMessage struct {
	Id      int    `json:"id"`
	Message string `json:"message"`
}

type ClientErrorMessage struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

func (c *WsClient) GetId() int {
	return c.id
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *WsClient) readPump() {
	defer func() {
		c.wsHub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			logging.Errorf(logging.ProtoWS, "Receive error: %s", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Errorf(logging.ProtoWS, "Unexpected error: %v", err)
			} else {
				c.conn.WriteJSON(ClientErrorMessage{
					ErrorCode: "receive-error",
					Message:   err.Error(),
				})
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		//c.hub.broadcast <- message
		c.wsHub.messageReceived <- &ReceivedMessage{
			Sender:  c,
			Message: message,
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *WsClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case message := <-c.SendMessage:
			json, err := json.Marshal(message)
			if err != nil {
				return
			}
			c.send <- json
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WsClient) RemoteAddrStr() string {
	if c.conn == nil {
		return "<nil>"
	}
	return c.conn.RemoteAddr().String()
}
