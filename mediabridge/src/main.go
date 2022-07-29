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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/adalkiran/distributed-inference/mediabridge/src/endpoint"
	logging "github.com/adalkiran/go-colorful-logging"
	"github.com/adalkiran/go-inventa"
)

var (
	endpointManager      *endpoint.EndpointManager
	inventaObj           *inventa.Inventa
	RPCCommandFnRegistry map[string]inventa.RPCCommandFn
)

var (
	SelfDescriptor      inventa.ServiceDescriptor
	SignalingDescriptor inventa.ServiceDescriptor
)

func main() {
	var err error
	hostname := os.Getenv("HOSTNAME")
	SelfDescriptor, err = inventa.ParseServiceFullId(fmt.Sprintf("svc:mb:%s", hostname))
	if err != nil {
		panic(err)
	}
	SignalingDescriptor, err = inventa.ParseServiceFullId("svc:sgn:")
	if err != nil {
		panic(err)
	}
	//See: https://codewithyury.com/golang-wait-for-all-goroutines-to-finish/
	//See: https://www.geeksforgeeks.org/using-waitgroup-in-golang/
	waitGroup := new(sync.WaitGroup)

	logging.Freef("", "Welcome to Distributed Inference Pipeline - MediaBridge Server!")
	logging.Freef("", "=================================")
	logging.Freef("", "This module acts as media bridge server and orchestrates other modules.")
	logging.LineSpacer(3)

	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		logging.Errorf(logging.ProtoAPP, "Invalid REDIS_PORT value: \"%s\"", os.Getenv("REDIS_PORT"))
		return
	}

	dockerHostIp := os.Getenv("MEDIABRIDGE_DOCKER_HOST_IP")

	udpPort, err := strconv.Atoi(os.Getenv("MEDIABRIDGE_UDP_PORT"))
	if err != nil {
		logging.Errorf(logging.ProtoAPP, "Invalid MEDIABRIDGE_UDP_PORT value: \"%s\"", os.Getenv("MEDIABRIDGE_UDP_PORT"))
		return
	}

	RPCCommandFnRegistry = map[string]inventa.RPCCommandFn{
		"sdp-offer-req":           rpcCommandSDPOfferReq,
		"sdp-accept-offer-answer": rpcCommandSDPAcceptOfferAnswer,
	}

	inventaObj = inventa.NewInventa(os.Getenv("REDIS_HOST"), redisPort, os.Getenv("REDIS_PASSWORD"), SelfDescriptor.ServiceType, SelfDescriptor.ServiceId, inventa.InventaRoleService, RPCCommandFnRegistry)
	_, err = inventaObj.Start()
	if err != nil {
		panic(err)
	}

	err = inventaObj.TryRegisterToOrchestrator(SignalingDescriptor.Encode(), 10, 3*time.Second)
	if err == nil {
		logging.Infof(logging.ProtoAPP, "Registered to signaling service as <u>%s</u>", SelfDescriptor.Encode())
	} else {
		logging.Errorf(logging.ProtoAPP, "Registration to signaling service was failed! Breaking down! %s", err)
		return
	}

	endpointManager = endpoint.NewEndpointManager(dockerHostIp, udpPort, inventaObj)
	waitGroup.Add(1)
	go endpointManager.Run(waitGroup)

	waitGroup.Wait()
}

func rpcCommandSDPOfferReq(req *inventa.RPCCallRequest) []string {
	participantId := req.Args[0]
	endpoint := endpointManager.EnsureEndpoint(participantId)
	logging.Infof(logging.ProtoAPP, "Starting up a connection for endpoint <u>%s</u>...", endpoint.ParticipantId)
	sdpOffer, err := endpoint.DoSignaling(endpointManager)
	if err != nil {
		logging.Errorf(logging.ProtoAPP, "Error while sending SDP Offer: %s", err)
		return req.ErrorResponse(err)
	}
	logging.Infof(logging.ProtoAPP, "SDP Offer sent for endpoint <u>%s</u>.\n%s\n\n", endpoint.ParticipantId, sdpOffer)
	bytesJson, _ := json.Marshal(sdpOffer)
	return []string{string(bytesJson)}
}

func rpcCommandSDPAcceptOfferAnswer(req *inventa.RPCCallRequest) []string {
	participantId := req.Args[0]
	sdpOfferAnswer := req.Args[1]
	endpoint, ok := endpointManager.Endpoints[participantId]
	if !ok {
		return req.ErrorResponse(fmt.Errorf("participant not found: %s", participantId))
	}
	err := endpoint.AcceptSDPOfferAnswer(sdpOfferAnswer)
	if err != nil {
		return req.ErrorResponse(err)
	}
	return []string{"succeeded"}
}
