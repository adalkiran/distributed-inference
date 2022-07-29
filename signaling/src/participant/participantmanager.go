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

package participant

import (
	"encoding/json"
	"fmt"
	"sync"

	logging "github.com/adalkiran/go-colorful-logging"
	"github.com/adalkiran/go-inventa"

	"github.com/adalkiran/distributed-inference/signaling/src/orchestration"
	"github.com/adalkiran/distributed-inference/signaling/src/signaling"
)

type ParticipantWsMessage struct {
	Participant *Participant
	Message     signaling.WsMessageContainer
}

type ParticipantManager struct {
	Participants                map[string]*Participant //key: ParticipantId
	participantsPerWsClient     map[int]*Participant    //key: WsClient.Id
	wsClientIdsPerParticipantId map[string][]int
	TenantManager               *TenantManager
	Orchestrator                *orchestration.Orchestrator

	inventa *inventa.Inventa

	SendMessage chan ParticipantWsMessage
	//ChanSdpOffer chan *sdp.SdpMessage
}

func NewParticipantManager(tenantManager *TenantManager, orchestrator *orchestration.Orchestrator, inventaObj *inventa.Inventa) *ParticipantManager {
	result := &ParticipantManager{
		Participants:            map[string]*Participant{},
		participantsPerWsClient: map[int]*Participant{},
		TenantManager:           tenantManager,
		Orchestrator:            orchestrator,
		inventa:                 inventaObj,
		SendMessage:             make(chan ParticipantWsMessage, 256),
	}
	result.Orchestrator.WsCommandFnRegistry["Join"] = result.WsCommandJoin
	result.Orchestrator.WsCommandFnRegistry["SdpOfferAnswer"] = result.WsCommandSdpOfferAnswer
	return result
}

func (m *ParticipantManager) CreateParticipant(tenantId string, wsClientId int) (*Participant, error) {
	tenant, ok := m.TenantManager.Tenants[tenantId]
	if !ok {
		return nil, fmt.Errorf("Tenant not found: %s", tenantId)
	}
	participant := NewParticipant(tenant, wsClientId)
	m.Participants[participant.Id] = participant
	m.participantsPerWsClient[wsClientId] = participant
	tenant.Participants[participant.Id] = participant
	//m.Agents[newConference.IceAgent.Ufrag] = newConference.IceAgent
	return participant, nil
}

func (m *ParticipantManager) GetParticipantByWsClientId(wsClientId int) (*Participant, bool) {
	result, ok := m.participantsPerWsClient[wsClientId]
	return result, ok
}

func (m *ParticipantManager) PrepareParticipant(participant *Participant) (string, error) {
	selectedMediaBridge := m.Orchestrator.SelectMediaBridgeForParticipant()
	if selectedMediaBridge == nil {
		return "", fmt.Errorf("there isn't any available mediabridge found")
	}
	sdpOffer, err := selectedMediaBridge.RequestSDPOffer(participant.Id)
	if err != nil {
		return "", fmt.Errorf("error while requesting SDP Offer from media bridge: %s, error: %s", selectedMediaBridge.SelfDescriptor.Encode(), err)
	}
	logging.Infof(logging.ProtoAPP, "Received sdpOffer from %s: %s", selectedMediaBridge.SelfDescriptor.Encode(), sdpOffer)
	participant.MediaBridge = selectedMediaBridge
	return sdpOffer, nil
}

func (m *ParticipantManager) Run(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for {
		select {
		case message := <-m.SendMessage:
			json, err := json.Marshal(message.Message)
			if err != nil {
				continue
			}
			m.Orchestrator.WsHub.Broadcast <- signaling.BroadcastMessage{
				IncludeClients: message.Participant.WsClientIds,
				Message:        json,
			}
		}
	}
}

func (m *ParticipantManager) WsCommandJoin(hub *signaling.WsHub, cmd *signaling.WsCommand) {
	tenantId := cmd.MessageData["tenantId"].(string)
	logging.Descf(logging.ProtoWS, "The <u>client %d</u> wanted to join as a participant of tenant <u>%s</u>.", cmd.WsClient.GetId(), tenantId)
	participant, err := m.CreateParticipant(tenantId, cmd.WsClient.GetId())
	if err != nil {
		cmd.WsClient.SendMessage <- signaling.NewWsMessageContainer("JoinError", err.Error())
		return
	}
	logging.Descf(logging.ProtoWS, "The client was joined as participant <u>%s</u> of tenant <u>%s</u>. Now we should assign an available media bridge, generate an SDP Offer including our UDP candidates (IP-port pairs) and send to the client via Signaling/WebSocket.", participant.Id, participant.Tenant.Name)
	sdpOffer, err := m.PrepareParticipant(participant)
	if err != nil {
		logging.Errorf(logging.ProtoAPP, "Error while joining: %s", err)
		cmd.WsClient.SendMessage <- signaling.NewWsMessageContainer("JoinError", err.Error())
		return
	}
	cmd.WsClient.SendMessage <- signaling.NewWsMessageContainer("SdpOffer", sdpOffer)
}

func (m *ParticipantManager) WsCommandSdpOfferAnswer(hub *signaling.WsHub, cmd *signaling.WsCommand) {
	participant, ok := m.GetParticipantByWsClientId(cmd.WsClient.GetId())
	if !ok {
		logging.Errorf(logging.ProtoWS, "WS Command SdpOfferAnswer was called before Join. Ignoring.")
		return
	}
	if participant.MediaBridge == nil {
		logging.Errorf(logging.ProtoWS, "WS Command SdpOfferAnswer was called before selecting a media bridge for participant. Ignoring.")
		return
	}
	sdpOfferAnswer := cmd.MessageData["sdp"].(string)
	_, err := participant.MediaBridge.AcceptSDPOfferAnswer(participant.Id, sdpOfferAnswer)
	if err != nil {
		logging.Errorf(logging.ProtoAPP, "Error while SDP Offer Answer: %s", err)
		cmd.WsClient.SendMessage <- signaling.NewWsMessageContainer("Error", err.Error())
		return
	}

}
