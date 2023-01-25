#   Copyright (c) 2022-present, Adil Alper DALKIRAN
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
#   ==============================================================================

import asyncio

import io
import string
from PIL import Image
import numpy as np

import onnxruntime

from yolox.data.data_augment import preproc as preprocess
from yolox.data.datasets import COCO_CLASSES
from yolox.utils import mkdir, multiclass_nms, demo_postprocess, vis

from inventa import Inventa, ServiceDescriptor

from redis import asyncio as aioredis

from .base_inference_task import BaseInferenceTask

STREAM_IMAGES = "images"
STREAM_PREDICTIONS = "predictions"
CONSUMER_GROUP_IMAGES = "cg:images"

class YoloxInferenceTask(BaseInferenceTask):
    def __init__(self, inventa: Inventa, SelfDescriptor: ServiceDescriptor):
        super().__init__(inventa, SelfDescriptor)

    def get_task_id(self) -> string:
        return "inf_yolox"

    def get_task_name(self) -> string:
        return "YOLOX Inference Task"
  
    def get_prediction(self, img_bytes,session):
        # See: https://github.com/Megvii-BaseDetection/YOLOX/blob/main/demo/ONNXRuntime/onnx_inference.py
        score_thr = 0.3
        origin_img = np.asarray(Image.open(io.BytesIO(img_bytes)))
        input_shape = (416, 416)
        img, ratio = preprocess(origin_img, input_shape)
        ort_inputs = {session.get_inputs()[0].name: img[None, :, :, :]}
        output = session.run(None, ort_inputs)
        predictions = demo_postprocess(output[0], input_shape, p6=False)[0]
        boxes = predictions[:, :4]
        scores = predictions[:, 4:5] * predictions[:, 5:]

        boxes_xyxy = np.ones_like(boxes)
        boxes_xyxy[:, 0] = boxes[:, 0] - boxes[:, 2]/2.
        boxes_xyxy[:, 1] = boxes[:, 1] - boxes[:, 3]/2.
        boxes_xyxy[:, 2] = boxes[:, 0] + boxes[:, 2]/2.
        boxes_xyxy[:, 3] = boxes[:, 1] + boxes[:, 3]/2.
        boxes_xyxy /= ratio
        dets = multiclass_nms(boxes_xyxy, scores, nms_thr=0.45, score_thr=score_thr)

        if dets is not None:
            final_boxes, final_scores, final_cls_inds = dets[:, :4], dets[:, 4], dets[:, 5]
            result = {
                "pcount": len(final_cls_inds),
                "res": origin_img.shape[0],
            }
            for i, cls_idx in enumerate(final_cls_inds):
                class_name = COCO_CLASSES[int(cls_idx)]
                score = final_scores[i]
                box = [str(int(item)) for item in final_boxes[i]]
                result["p" + str(i)] = "|".join([class_name, str(score), ";".join(box)])
            return result
            #origin_img = vis(origin_img, final_boxes, final_scores, final_cls_inds,
            #                 conf=score_thr, class_names=COCO_CLASSES)



        return {
            "pcount": 0
        }
    
    async def autoclaim_stream(self, self_consumer_name):
        # See: https://redis.io/commands/xautoclaim/
        # Command pattern:
        # XAUTOCLAIM key group consumer min-idle-time start [COUNT count] [JUSTID]
        claimed_messages_response = await self.inventa.Client.execute_command(
            b"XAUTOCLAIM", STREAM_IMAGES, CONSUMER_GROUP_IMAGES, self_consumer_name, 5000, 0, "COUNT", 10)
        claimed_messages = claimed_messages_response[1]
        result = []
        for message in claimed_messages:
            message_id, data = message
            item_data = {}
            for i in range(0, len(data), 2):
                item_data[data[i]] = data[i + 1]
            result.append([message_id, item_data])
        return result
    

    async def process_images_messages(self, messages, inventa: Inventa, session):
        for message in messages:
            with self.metrics_inference_time.time():
                message_id, data = message
                participant_id = data[b"participantId"]
                timestamp = data[b"timestamp"]
                img_bytes = data[b"img"]
                # print("REDIS ID: ", message_id, ", participantId: ", participant_id, ", timestamp: ", timestamp)
                predictions = self.get_prediction(img_bytes, session)
                meta_data = {
                    "participantId": participant_id,
                    "timestamp": timestamp,
                }
                stream_data = {**meta_data, **predictions}
                # print("Predictions: ", stream_data)
                #See: https://redis.io/commands/xadd/
                await inventa.Client.xadd(STREAM_PREDICTIONS, stream_data, maxlen=100)
                await inventa.Client.xack(STREAM_IMAGES, CONSUMER_GROUP_IMAGES, message_id)
                self.metrics_inference_frame_count.inc()
                await asyncio.sleep(0)

    async def run(self):
        print(f"{self.get_task_name()} will run on {onnxruntime.get_device()}")
        session = onnxruntime.InferenceSession("/home/yolox_nano.onnx", providers=['CUDAExecutionProvider', 'CPUExecutionProvider'])
        # See: https://redis-py-doc.readthedocs.io/en/master/
        # See: https://developpaper.com/detailed-explanation-of-redis-stream-type-commands/
        try:
            await self.inventa.Client.xgroup_create(STREAM_IMAGES, CONSUMER_GROUP_IMAGES, id="$", mkstream=True)
        except:
            pass
        first_claimed = False
        claimed_messages = []
        self_consumer_name = self.SelfDescriptor.Encode()
        while not first_claimed or len(claimed_messages): 
            #claimed_messages = await self.autoclaim_stream(self_consumer_name)
            # See: https://redis.io/commands/xautoclaim/
            claimed_messages_response = await self.inventa.Client.xautoclaim(STREAM_IMAGES, CONSUMER_GROUP_IMAGES, self_consumer_name, min_idle_time=5000, start_id=0, count=10)
            claimed_messages = claimed_messages_response[1]
            first_claimed = True
            if claimed_messages:
                asyncio.ensure_future(self.process_images_messages(claimed_messages, self.inventa, session))
            await asyncio.sleep(0)
        
        while True:
            try:
                # See: https://huogerac.hashnode.dev/using-redis-stream-with-python
                # See: https://redis.io/commands/xread/
                resp = await self.inventa.Client.xreadgroup(CONSUMER_GROUP_IMAGES, self_consumer_name,
                    {STREAM_IMAGES: ">"}, count=1, block=10000
                )
                if resp:
                    key, messages = resp[0]
                    asyncio.ensure_future(self.process_images_messages(messages, self.inventa, session))

            except aioredis.ConnectionError as e:
                print(f"ERROR REDIS CONNECTION: {e}")
                await asyncio.sleep(.1)
            await asyncio.sleep(.01)

