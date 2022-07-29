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

package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	logging "github.com/adalkiran/go-colorful-logging"
	"github.com/adalkiran/go-inventa"

	"github.com/adalkiran/distributed-inference/signaling/src/orchestration"
	"github.com/adalkiran/distributed-inference/signaling/src/participant"
	"github.com/adalkiran/distributed-inference/signaling/src/predictions"
	"github.com/adalkiran/distributed-inference/signaling/src/signaling"
)

var (
	SelfDescriptor inventa.ServiceDescriptor

	orchestrator       *orchestration.Orchestrator
	tenantManager      *participant.TenantManager
	participantManager *participant.ParticipantManager
	inventaObj         *inventa.Inventa
)

func main() {
	var err error
	SelfDescriptor, err = inventa.ParseServiceFullId("svc:sgn:")
	if err != nil {
		panic(err)
	}

	//See: https://codewithyury.com/golang-wait-for-all-goroutines-to-finish/
	//See: https://www.geeksforgeeks.org/using-waitgroup-in-golang/
	waitGroup := new(sync.WaitGroup)

	logging.Freef("", "Welcome to Distributed Inference Pipeline - Signaling Server!")
	logging.Freef("", "=================================")
	logging.Freef("", "This module acts as signaling server and orchestrates other modules.")
	logging.LineSpacer(3)

	orchestrator = orchestration.NewOrchestrator()

	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		logging.Errorf(logging.ProtoAPP, "Invalid REDIS_PORT value: \"%s\"", os.Getenv("REDIS_PORT"))
		return
	}

	inventaObj = inventa.NewInventa(os.Getenv("REDIS_HOST"), redisPort, os.Getenv("REDIS_PASSWORD"), SelfDescriptor.ServiceType, SelfDescriptor.ServiceId, inventa.InventaRoleOrchestrator, orchestrator.RPCCommandFnRegistry)

	orchestrator.Init(inventaObj)

	tenantManager = participant.NewTenantManager(inventaObj)
	participantManager = participant.NewParticipantManager(tenantManager, orchestrator, inventaObj)
	waitGroup.Add(1)
	go participantManager.Run(waitGroup)

	InitDatabase(tenantManager)
	tenantManager.InitFromDatabase()

	httpServer, err := signaling.NewHttpServer(fmt.Sprintf(":%d", 80), orchestrator.WsCommandFnRegistry)
	if err != nil {
		logging.Errorf(logging.ProtoAPP, "Http Server error: %s", err)
	}
	orchestrator.WsHub = httpServer.WsHub
	waitGroup.Add(1)
	go httpServer.Run(waitGroup)

	predictionListener := predictions.NewPredictionListener(participantManager, inventaObj, SelfDescriptor.Encode())

	waitGroup.Add(1)
	go predictionListener.Run(waitGroup)

	waitGroup.Wait()
}
