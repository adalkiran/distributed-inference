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
	"time"

	logging "github.com/adalkiran/go-colorful-logging"
	"github.com/adalkiran/go-inventa"
)

type MediaBridgeEndpoint struct {
	ParticipantId      string
	ICEConnectionState string
}

type MediaBridgeModule struct {
	inventa.ServiceConsumer

	Endpoints map[string]*MediaBridgeEndpoint
}

func NewMediaBridgeModule(id string, inventaObj *inventa.Inventa) *MediaBridgeModule {
	return &MediaBridgeModule{
		ServiceConsumer: inventa.ServiceConsumer{
			SelfDescriptor: inventa.ServiceDescriptor{
				ServiceType: "mb",
				ServiceId:   id,
			},
			Inventa: inventaObj,
		},
		Endpoints: map[string]*MediaBridgeEndpoint{},
	}
}

func (m *MediaBridgeModule) RequestSDPOffer(participantId string) (string, error) {
	response, err := m.Inventa.CallSync(m.SelfDescriptor.Encode(), "sdp-offer-req", []string{participantId}, 3*time.Second)
	if err != nil {
		return "", err
	}
	return response[0], nil
}

func (m *MediaBridgeModule) AcceptSDPOfferAnswer(participantId string, sdpOfferAnswer string) (string, error) {
	args := []string{participantId, sdpOfferAnswer}
	response, err := m.Inventa.CallSync(m.SelfDescriptor.Encode(), "sdp-accept-offer-answer", args, 3*time.Second)
	if err != nil {
		return "", err
	}
	m.Endpoints[participantId] = &MediaBridgeEndpoint{
		ParticipantId:      participantId,
		ICEConnectionState: "new",
	}
	return response[0], nil
}

func (m *MediaBridgeModule) EndpointICEConnStateChange(participantId string, newConnectionState string, oldConnectionState string) error {
	endpoint, ok := m.Endpoints[participantId]
	if !ok {
		return fmt.Errorf("participant not found: %s", participantId)
	}
	endpoint.ICEConnectionState = newConnectionState
	logging.Infof(logging.ProtoAPP, "Endpoint <u>%s</u> on media bridge <u>%s</u> ICE connection state changed from <u>%s</u> to <u>%s</u>", participantId, m.SelfDescriptor.Encode(), oldConnectionState, newConnectionState)
	if newConnectionState == "failed" {
		delete(m.Endpoints, participantId)
	}
	return nil
}
