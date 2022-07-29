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
	"bytes"
	"fmt"
	"image/jpeg"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/xlab/libvpx-go/vpx"

	logging "github.com/adalkiran/go-colorful-logging"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

type Endpoint struct {
	ParticipantId              string
	PeerConnection             *webrtc.PeerConnection
	ICEConnectionState         string
	OnICEConnectionStateChange func(newConnectionState string, oldConnectionState string)
}

type SignalingResult struct {
	SDP string `json:"sdp"`
}

const (
	STREAM_IMAGES = "images"
)

func NewEndpoint(participantId string) *Endpoint {
	result := &Endpoint{
		ParticipantId:      participantId,
		ICEConnectionState: "new",
	}
	return result
}

func (e *Endpoint) DoSignaling(endpointManager *EndpointManager) (*SignalingResult, error) {
	//See: https://github.com/pion/webrtc/blob/157220e800257ee4090f181e7edcca6435adb9f2/examples/save-to-disk/main.go
	//See: https://github.com/pion/webrtc/blob/master/examples/pion-to-pion/offer/main.go
	//See: https://github.com/pion/webrtc/blob/master/examples/ice-single-port/main.go
	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := endpointManager.WebrtcApi.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	/*
		// Allow us to receive 1 audio track, and 1 video track
		if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
			return "", err
		} else if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
			return "", err
		}
	*/

	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}

	// When an ICE candidate is available send to the other Pion instance
	// the other Pion instance will add this candidate by calling AddICECandidate
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			logging.Infof(logging.ProtoAPP, "OnICECandidate desc nil: %s", c)
			pendingCandidates = append(pendingCandidates, c)
		} else {
			logging.Infof(logging.ProtoAPP, "OnICECandidate desc not nil: %s", c)
		}
	})

	// Set a handler for when a new remote track starts, this handler saves buffers to disk as
	// an ivf file, since we could have multiple video tracks we provide a counter.
	// In your application this is where you would handle/process video
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if errSend != nil {
					fmt.Println(errSend)
				}
			}
		}()

		codec := track.Codec()
		if strings.EqualFold(codec.MimeType, webrtc.MimeTypeOpus) {
			fmt.Println("Got Opus track, saving to disk as output.opus (48 kHz, 2 channels)")
			//saveToDisk(oggFile, track)
		} else if strings.EqualFold(codec.MimeType, webrtc.MimeTypeVP8) {
			fmt.Println("Got VP8 track, starting depacketizer...")
			go e.processVP8Track(track, endpointManager)
			//saveToDisk(ivfFile, track)
		}
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		logging.Infof(logging.ProtoAPP, "Connection State has changed to <u>%s</u> for participant <u>%s</u>", connectionState.String(), e.ParticipantId)
		if e.OnICEConnectionStateChange != nil {
			oldState := e.ICEConnectionState
			e.ICEConnectionState = connectionState.String()
			e.OnICEConnectionStateChange(e.ICEConnectionState, oldState)
		}
	})

	offer, err := peerConnection.CreateOffer(&webrtc.OfferOptions{})

	if err != nil {
		return nil, err
	}

	// Set the remote SessionDescription
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		return nil, err
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	offer, err = peerConnection.CreateOffer(&webrtc.OfferOptions{})

	if err != nil {
		return nil, err
	}

	e.PeerConnection = peerConnection

	return &SignalingResult{
		SDP: offer.SDP,
	}, nil
}

func (e *Endpoint) AcceptSDPOfferAnswer(sdpOfferAnswer string) error {
	return e.PeerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdpOfferAnswer,
	})
}

func (e *Endpoint) processVP8Track(track *webrtc.TrackRemote, endpointManager *EndpointManager) {
	//See: https://gist.github.com/mholbergerNIMBL/e5d17491a8cad621d53c6ed1b505ab7a#file-main-go-L6
	//See: https://stackoverflow.com/questions/68859120/how-to-convert-vp8-interframe-into-image-with-pion-webrtc
	sampleBuilder := samplebuilder.New(20000, &codecs.VP8Packet{}, track.Codec().ClockRate)
	decoder := vpx.DecoderIfaceVP8()
	ctx := vpx.NewCodecCtx()

	err := vpx.Error(vpx.CodecDecInitVer(ctx, decoder, nil, 0, vpx.DecoderABIVersion))
	if err != nil {
		logging.Warningf("RTP", "%s", err)
		return
	}

	fileCount := 0

	lastCount := fileCount
	lastTime := time.Now()

	for {
		now := time.Now()
		if now.After(lastTime.Add(1 * time.Second)) {
			logging.Descf("RTP", "Processed frame count for <u>%s</u>: <u>%d</u>", e.ParticipantId, fileCount-lastCount)
			lastCount = fileCount
			lastTime = now
		}
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			panic(err)
		}
		sampleBuilder.Push(rtpPacket)

		samplePop := sampleBuilder.Pop()
		if samplePop == nil {
			//a: logging.Descf("RTP", "Packet recieved, NO FRAME size: %d", len(rtpPacket.Payload))
			continue
		}

		dataSize := uint32(len(samplePop.Data))

		err = vpx.Error(vpx.CodecDecode(ctx, string(samplePop.Data), dataSize, nil, 0))
		if err != nil {
			log.Println("[WARN]", err)
			continue
		}

		var iter vpx.CodecIter
		img := vpx.CodecGetFrame(ctx, &iter)
		if img != nil {
			img.Deref()
			fileCount++

			buffer := new(bytes.Buffer)
			if err = jpeg.Encode(buffer, img.ImageYCbCr(), nil); err != nil {
				//  panic(err)
				fmt.Printf("jpeg Encode Error: %s\r\n", err)
				continue
			}

			//See: https://redis.io/commands/xadd/
			err := endpointManager.Inventa.Client.XAdd(endpointManager.Inventa.Ctx, &redis.XAddArgs{
				Stream: STREAM_IMAGES,
				MaxLen: 100,
				ID:     "",
				Values: map[string]interface{}{
					"participantId": e.ParticipantId,
					"timestamp":     rtpPacket.Timestamp,
					"img":           buffer.Bytes(),
				},
			}).Err()

			if err != nil {
				logging.Errorf("RTP", "Redis XADD Error: %s", err)
				continue
			}
		}
	}
}
