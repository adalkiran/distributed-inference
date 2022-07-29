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

package endpoint

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/adalkiran/go-inventa"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

type EndpointManager struct {
	Endpoints    map[string]*Endpoint //key: ParticipantId
	WebrtcApi    *webrtc.API
	Inventa      *inventa.Inventa
	dockerHostIp string
	udpPort      int
	udpListener  *net.UDPConn
}

func NewEndpointManager(dockerHostIp string, udpPort int, inventaObj *inventa.Inventa) *EndpointManager {
	result := &EndpointManager{
		Endpoints:    map[string]*Endpoint{},
		Inventa:      inventaObj,
		dockerHostIp: dockerHostIp,
		udpPort:      udpPort,
	}

	return result
}

func (m *EndpointManager) Run(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	//See: https://github.com/pion/webrtc/blob/master/examples/ice-single-port/main.go
	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: m.udpPort,
	})
	if err != nil {
		panic(err)
	}

	m.udpListener = udpListener

	fmt.Printf("Listening for WebRTC traffic at %s\n", udpListener.LocalAddr())

	// Create a SettingEngine, this allows non-standard WebRTC behavior
	settingEngine := webrtc.SettingEngine{}

	// Configure our SettingEngine to use our UDPMux. By default a PeerConnection has
	// no global state. The API+SettingEngine allows the user to share state between them.
	// In this case we are sharing our listening port across many.
	settingEngine.SetICEUDPMux(webrtc.NewICEUDPMux(nil, udpListener))

	if m.dockerHostIp != "" {
		settingEngine.SetNAT1To1IPs([]string{m.dockerHostIp}, webrtc.ICECandidateTypeHost)
	}

	mediaEngine, err := m.createMediaEngine()
	if err != nil {
		panic(err)
	}
	interceptorRegistry, err := m.createInterceptorRegistry(mediaEngine)
	if err != nil {
		panic(err)
	}
	// Create a new API using our SettingEngine
	m.WebrtcApi = webrtc.NewAPI(
		webrtc.WithSettingEngine(settingEngine),
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
	)
	select {}
}

func (m *EndpointManager) createMediaEngine() (*webrtc.MediaEngine, error) {
	//See: https://github.com/pion/webrtc/blob/157220e800257ee4090f181e7edcca6435adb9f2/examples/save-to-disk/main.go
	// Create a MediaEngine object to configure the supported codec
	me := &webrtc.MediaEngine{}

	// Setup the codecs you want to use.
	// We'll use a VP8 and Opus but you can also define your own
	if err := me.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}
	if err := me.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, err
	}
	return me, nil
}

func (m *EndpointManager) createInterceptorRegistry(me *webrtc.MediaEngine) (*interceptor.Registry, error) {
	//See: https://github.com/pion/webrtc/blob/157220e800257ee4090f181e7edcca6435adb9f2/examples/save-to-disk/main.go
	// Create a InterceptorRegistry. This is the user configurable RTP/RTCP Pipeline.
	// This provides NACKs, RTCP Reports and other features. If you use `webrtc.NewPeerConnection`
	// this is enabled by default. If you are manually managing You MUST create a InterceptorRegistry
	// for each PeerConnection.
	i := &interceptor.Registry{}

	// Use the default set of Interceptors
	if err := webrtc.RegisterDefaultInterceptors(me, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (m *EndpointManager) EnsureEndpoint(participantId string) *Endpoint {
	endpoint, ok := m.Endpoints[participantId]
	if !ok {
		endpoint = NewEndpoint(participantId)
		endpoint.OnICEConnectionStateChange = func(newConnectionState string, oldConnectionState string) {
			m.Inventa.CallSync(
				m.Inventa.OrchestratorDescriptor.Encode(),
				"mb-ice-conn-state-change",
				[]string{endpoint.ParticipantId, newConnectionState, oldConnectionState},
				3*time.Second)
		}
		m.Endpoints[endpoint.ParticipantId] = endpoint
	}
	return endpoint
}
