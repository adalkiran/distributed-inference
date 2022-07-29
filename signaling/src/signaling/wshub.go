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
	"encoding/json"
	"sync"

	logging "github.com/adalkiran/go-colorful-logging"
)

// See: https://github.com/gorilla/websocket/tree/master/examples/chat

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type WsHub struct {
	sync.Mutex

	WsCommandFnRegistry map[string]WsCommandFn

	maxClientId int

	// Registered clients.
	clients map[int]*WsClient

	// Inbound messages from the clients.
	messageReceived chan *ReceivedMessage

	Broadcast chan BroadcastMessage

	// Register requests from the clients.
	register chan *WsClient

	// Unregister requests from clients.
	unregister chan *WsClient

	processMessage chan *WsCommand
}

type BroadcastMessage struct {
	Message        []byte
	IncludeClients []int
	ExcludeClients []int
}

type WsMessageContainer struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewWsMessageContainer(type_ string, data interface{}) WsMessageContainer {
	return WsMessageContainer{
		Type: type_,
		Data: data,
	}
}

func newWsHub(wsCommandFnRegistry map[string]WsCommandFn) *WsHub {
	return &WsHub{
		WsCommandFnRegistry: wsCommandFnRegistry,
		messageReceived:     make(chan *ReceivedMessage),
		Broadcast:           make(chan BroadcastMessage),
		register:            make(chan *WsClient),
		unregister:          make(chan *WsClient),
		processMessage:      make(chan *WsCommand, 10),
		clients:             make(map[int]*WsClient),
	}
}

func processBroadcastMessage(h *WsHub, broadcastMessage BroadcastMessage) {
	if len(broadcastMessage.IncludeClients) > 0 {
		for _, clientId := range broadcastMessage.IncludeClients {
			client, ok := h.clients[clientId]
			if !ok {
				continue
			}
			client.send <- broadcastMessage.Message
		}
		return
	}
	for clientId, client := range h.clients {
		if broadcastMessage.ExcludeClients != nil {
			var found = false
			for _, item := range broadcastMessage.ExcludeClients {
				if item == clientId {
					found = true
					break
				}
			}
			if found {
				continue
			}
		}
		select {
		case client.send <- broadcastMessage.Message:
		default:
			close(client.send)
			close(client.SendMessage)
			delete(h.clients, client.id)
		}
	}
}

/*
func writeContainerJSON(client *WsClient, messageType string, messageData interface{}) {
	client.conn.WriteJSON(WsMessageContainer{
		Type: messageType,
		Data: messageData,
	})
}
*/
func (h *WsHub) run() {
	for {
		select {
		case client := <-h.register:
			h.Lock()
			h.maxClientId++
			client.id = h.maxClientId
			h.clients[client.id] = client
			h.Unlock()
			logging.Infof(logging.ProtoWS, "A new client connected: <u>client %d</u> (from <u>%s</u>)", client.id, client.conn.RemoteAddr())
			logging.Descf(logging.ProtoWS, "Sending welcome message via WebSocket. The client is informed with client ID given by the signaling server.")
			client.SendMessage <- NewWsMessageContainer("Welcome", ClientWelcomeMessage{
				Id:      client.id,
				Message: "Welcome!",
			})
			logging.Descf(logging.ProtoWS, "Sent")
		case client := <-h.unregister:
			h.Lock()
			if _, ok := h.clients[client.id]; ok {
				delete(h.clients, client.id)
				close(client.send)
				logging.Infof(logging.ProtoWS, "Client disconnected: <u>client %d</u> (from <u>%s</u>)", client.id, client.conn.RemoteAddr())
			}
			h.Unlock()
		case broadcastMessage := <-h.Broadcast:
			processBroadcastMessage(h, broadcastMessage)
		case wsCommand := <-h.processMessage:
			wsCommand.Execute(h)
		case receivedMessage := <-h.messageReceived:
			var messageObj map[string]interface{}
			json.Unmarshal(receivedMessage.Message, &messageObj)

			logging.Infof(logging.ProtoWS, "Message received from <u>client %d</u> type <u>%s</u>", receivedMessage.Sender.id, messageObj["type"])
			wsCommandFn, ok := h.WsCommandFnRegistry[messageObj["type"].(string)]
			if !ok {
				logging.Errorf(logging.ProtoWS, "Unknown message type: %s", messageObj["type"])
				continue
			}
			w := NewWsCommand(messageObj["data"].(map[string]interface{}), receivedMessage.Sender, wsCommandFn)
			h.processMessage <- w
		}
	}
}
