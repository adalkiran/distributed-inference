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
	"github.com/adalkiran/go-inventa"
)

type TenantManager struct {
	Tenants map[string]*Tenant //key: TenantId

	Inventa *inventa.Inventa
}

func NewTenantManager(inventaObj *inventa.Inventa) *TenantManager {
	result := &TenantManager{
		Tenants: map[string]*Tenant{},
		Inventa: inventaObj,
	}
	return result
}

func (m *TenantManager) EnsureTenant(id string, name string) *Tenant {
	tenant, ok := m.Tenants[id]
	if !ok {
		tenant := NewTenant(id, name)
		m.Tenants[tenant.Id] = tenant
		m.SaveTenant(tenant)
	}
	return tenant
}

func (m *TenantManager) SaveTenant(entity *Tenant) {
	m.Inventa.Client.HSet(m.Inventa.Ctx, "TENANTS", entity.Id, entity.Name)
}

func (m *TenantManager) GetTenant(id string) *Tenant {
	//response := r.client.HGet(r.ctx, "TENANTS", id)
	return nil
}

func (m *TenantManager) GetAllTenants() []*Tenant {
	response := m.Inventa.Client.HGetAll(m.Inventa.Ctx, "TENANTS")
	result := make([]*Tenant, 0)
	for key, val := range response.Val() {
		result = append(result, NewTenant(key, val))
	}
	return result
}

func (m *TenantManager) InitFromDatabase() {
	for _, tenant := range m.GetAllTenants() {
		m.Tenants[tenant.Id] = tenant
	}
}

/*
func (m *ConferenceManager) Run(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for {
		select {
		case sdpOffer := <-m.ChanSdpOffer:
			conference, ok := m.Conferences[sdpOffer.ConferenceName]
			if !ok {
				logging.Warningf(logging.ProtoSDP, "Conference not found: <u>%s</u>, ignoring SdpOffer\n", sdpOffer.ConferenceName)
				continue
			}
			for _, sdpMediaItem := range sdpOffer.MediaItems {
				conference.IceAgent.EnsureSignalingMediaComponent(sdpMediaItem.Ufrag, sdpMediaItem.Pwd, sdpMediaItem.FingerprintHash)
			}
			logging.Descf(logging.ProtoSDP, "We processed incoming SDP, notified the conference's ICE Agent object (SignalingMediaComponents) about client (media) components' ufrag, pwd and fingerprint hash in the SDP. The server knows some metadata about the UDP packets will come in future. Now we are waiting for a STUN Binding Request packet via UDP, with server Ufrag <u>%s</u> from the client!", sdpOffer.MediaItems[0].Ufrag)
		}
	}
}
*/
