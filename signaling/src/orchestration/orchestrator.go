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

package orchestration

import (
	"fmt"
	"math"
	"sync"

	"github.com/adalkiran/distributed-inference/signaling/src/signaling"
	logging "github.com/adalkiran/go-colorful-logging"
	"github.com/adalkiran/go-inventa"
)

type Orchestrator struct {
	sync.Mutex
	MediaBridgeModules map[string]*MediaBridgeModule

	Inventa              *inventa.Inventa
	RPCCommandFnRegistry map[string]inventa.RPCCommandFn

	WsHub               *signaling.WsHub
	WsCommandFnRegistry map[string]signaling.WsCommandFn
}

func NewOrchestrator() *Orchestrator {
	result := &Orchestrator{
		MediaBridgeModules:  map[string]*MediaBridgeModule{},
		WsCommandFnRegistry: map[string]signaling.WsCommandFn{},
	}
	result.RPCCommandFnRegistry = map[string]inventa.RPCCommandFn{
		"mb-ice-conn-state-change": result.rpcCommandMbICEConnStateChange,
	}
	return result
}

func (o *Orchestrator) SelectMediaBridgeForParticipant() *MediaBridgeModule {
	o.Lock()
	defer o.Unlock()
	var selectedBridge *MediaBridgeModule
	minEndpointCount := math.MaxInt

	for _, item := range o.MediaBridgeModules {
		if len(item.Endpoints) <= minEndpointCount {
			minEndpointCount = len(item.Endpoints)
			selectedBridge = item
		}
	}
	return selectedBridge
}

func (o *Orchestrator) Init(inventaObj *inventa.Inventa) {
	o.Inventa = inventaObj
	o.Inventa.OnServiceRegistering = o.ServiceRegisteringHandler
	o.Inventa.OnServiceUnregistering = o.ServiceUnregisteringHandler
	_, err := o.Inventa.Start()
	if err != nil {
		panic(err)
	}
}

func (o *Orchestrator) ServiceRegisteringHandler(serviceDescriptor inventa.ServiceDescriptor) error {
	o.Lock()
	defer o.Unlock()
	switch serviceDescriptor.ServiceType {
	case "mb":
		o.MediaBridgeModules[serviceDescriptor.ServiceId] = NewMediaBridgeModule(serviceDescriptor.ServiceId, o.Inventa)
		logging.Infof(logging.ProtoAPP, "Media Bridge module has been registered as <u>%s</u>", serviceDescriptor.Encode())
		return nil
	case "inf":
		logging.Infof(logging.ProtoAPP, "Inference module has been registered as <u>%s</u>", serviceDescriptor.Encode())
		return nil
	default:
		logging.Errorf(logging.ProtoAPP, "Unknown service type to register: <u>%s</u>. Raw args: <u>%s</u>", serviceDescriptor.ServiceType, serviceDescriptor.Encode())
		return fmt.Errorf("unknown service type to register: %s", serviceDescriptor.ServiceType)
	}
}

func (o *Orchestrator) ServiceUnregisteringHandler(serviceDescriptor inventa.ServiceDescriptor, isZombie bool) error {
	o.Lock()
	defer o.Unlock()
	switch serviceDescriptor.ServiceType {
	case "mb":
		mb := o.MediaBridgeModules[serviceDescriptor.ServiceId]
		if mb == nil {
			return nil
		}
		delete(o.MediaBridgeModules, serviceDescriptor.ServiceId)
		var warningFormat string
		if isZombie {
			warningFormat = "Media Bridge module <u>%s</u> is not alive anymore, it has been unregistered."
		} else {
			warningFormat = "Media Bridge module has been unregistered: <u>%s</u>"
		}
		logging.Warningf(logging.ProtoAPP, warningFormat, serviceDescriptor.Encode())
		return nil
	default:
		logging.Errorf(logging.ProtoAPP, "Unknown service type to unregister: <u>%s</u>. Raw args: <u>%s</u>", serviceDescriptor.ServiceType, serviceDescriptor.Encode())
		return fmt.Errorf("unknown service type to unregister: %s", serviceDescriptor.ServiceType)
	}
}

func (o *Orchestrator) rpcCommandMbICEConnStateChange(req *inventa.RPCCallRequest) []string {
	if req.FromService.ServiceType != "mb" {
		return req.ErrorResponse(fmt.Errorf("not compatible FromService.ServiceType for rpcCommandMbICEConnStateChange: %s", req.FromService.ServiceType))
	}
	mb, ok := o.MediaBridgeModules[req.FromService.ServiceId]
	if !ok {
		return req.ErrorResponse(fmt.Errorf("unknown MediaBridge id for rpcCommandMbICEConnStateChange: %s", req.FromService.ServiceId))
	}
	participantId := req.Args[0]
	newConnectionState := req.Args[1]
	oldConnectionState := req.Args[2]
	err := mb.EndpointICEConnStateChange(participantId, newConnectionState, oldConnectionState)
	if err != nil {
		return req.ErrorResponse(err)
	}
	return []string{"succeeded"}
}
