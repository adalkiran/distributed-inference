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
	"net/http"
	"sync"

	logging "github.com/adalkiran/go-colorful-logging"
)

type HttpServer struct {
	HttpServerAddr string
	WsHub          *WsHub
}

func NewHttpServer(httpServerAddr string, wsCommandFnRegistry map[string]WsCommandFn) (*HttpServer, error) {
	wsHub := newWsHub(wsCommandFnRegistry)

	httpServer := &HttpServer{
		HttpServerAddr: httpServerAddr,
		WsHub:          wsHub,
	}
	http.HandleFunc("/", httpServer.serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		httpServer.serveWs(w, r)
	})

	return httpServer, nil
}

func (s *HttpServer) Run(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	go s.WsHub.run()
	logging.Infof(logging.ProtoWS, "WebSocket Server started on <u>%s</u>", s.HttpServerAddr)
	logging.Descf(logging.ProtoWS, "Clients should make first contact with this WebSocket (the Signaling part)")
	err := http.ListenAndServe(s.HttpServerAddr, nil)
	if err != nil {
		panic(err)
	}
}

func (s *HttpServer) serveHome(w http.ResponseWriter, r *http.Request) {
	logging.Infof(logging.ProtoHTTP, "Request: <u>%s</u>", r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

// serveWs handles websocket requests from the peer.
func (s *HttpServer) serveWs(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Errorf(logging.ProtoHTTP, "Error: %s", err)
		return
	}
	client := &WsClient{wsHub: s.WsHub, conn: conn, send: make(chan []byte, 256), SendMessage: make(chan WsMessageContainer, 256)}
	client.wsHub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
