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

package predictions

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	goredis "github.com/go-redis/redis/v9"

	"github.com/adalkiran/distributed-inference/signaling/src/participant"
	"github.com/adalkiran/distributed-inference/signaling/src/signaling"
	logging "github.com/adalkiran/go-colorful-logging"
	"github.com/adalkiran/go-inventa"
)

type PredictionListener struct {
	participantManager *participant.ParticipantManager
	Inventa            *inventa.Inventa
	selfConsumerName   string
}

var (
	STREAM_PREDICTIONS         = "predictions"
	CONSUMER_GROUP_PREDICTIONS = "cg:PREDICTIONS"
)

func NewPredictionListener(participantManager *participant.ParticipantManager, inventaObj *inventa.Inventa, selfConsumerName string) *PredictionListener {
	result := &PredictionListener{
		participantManager: participantManager,
		Inventa:            inventaObj,
		selfConsumerName:   selfConsumerName,
	}
	return result
}

func (m *PredictionListener) Run(waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for {
		m.Inventa.Client.XGroupCreateMkStream(m.Inventa.Ctx, STREAM_PREDICTIONS, CONSUMER_GROUP_PREDICTIONS, "$").Err()

		entries, err := m.Inventa.Client.XReadGroup(m.Inventa.Ctx, &goredis.XReadGroupArgs{
			Group:    CONSUMER_GROUP_PREDICTIONS,
			Consumer: m.selfConsumerName,
			Streams:  []string{STREAM_PREDICTIONS, ">"},
			Count:    1,
			Block:    2 * time.Second,
		}).Result()

		if err != nil {
			if err.Error() != "redis: nil" {
				logging.Errorf(logging.ProtoAPP, "Redis XREADGROUP Error: %s", err)
			}
			continue
		}
		m.processPredictionMessages(entries[0].Messages)
	}
}

func (m *PredictionListener) processPredictionMessages(messages []goredis.XMessage) {
	for _, message := range messages {
		participantId := message.Values["participantId"].(string)
		timestamp := message.Values["timestamp"].(string)
		predictionCount, err := strconv.Atoi(message.Values["pcount"].(string))
		if err != nil {
			continue
		}
		participantObj, ok := m.participantManager.Participants[participantId]
		if !ok {
			continue
		}
		m.participantManager.SendMessage <- participant.ParticipantWsMessage{
			Participant: participantObj,
			Message: signaling.WsMessageContainer{
				Type: "Prediction",
				Data: message.Values,
			},
		}
		fmt.Printf("participantId: %s, timestamp: %s, predictionCount: %d\n", participantId, timestamp, predictionCount)
	}
}
