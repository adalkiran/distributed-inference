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

type WsCommandFn func(hub *WsHub, cmd *WsCommand)

type WsCommand struct {
	MessageData map[string]interface{}
	WsClient    *WsClient
	Fn          WsCommandFn
}

func NewWsCommand(messageData map[string]interface{}, wsClient *WsClient, fn WsCommandFn) *WsCommand {
	return &WsCommand{
		MessageData: messageData,
		WsClient:    wsClient,
		Fn:          fn,
	}
}

func (c *WsCommand) Execute(hub *WsHub) {
	c.Fn(hub, c)
}
