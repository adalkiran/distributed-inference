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
from os import environ
import traceback
import string

from prometheus_client import start_http_server, Summary, Counter

from tasks import ExportedInferenceTasks

environ.update([("PYTHONASYNCIODEBUG", "1")])

from inventa import Inventa, InventaRole, ServiceDescriptor

hostname = environ.get("HOSTNAME")
SelfDescriptor = ServiceDescriptor.ParseServiceFullId(f"svc:inf:{hostname}")
SignalingDescriptor = ServiceDescriptor.ParseServiceFullId("svc:sgn:")

def create_metrics_for_task(task_id: string):
    return {
        "inference_time": Summary(f"inference_time", f"Duration of frame processing", ["task"]).labels(task=task_id),
        "inference_frame_count": Counter(f"inference_frame_count", f"Total number of processed frames", ["task"]).labels(task=task_id)
    }


def connect_to_redis() -> Inventa:
    hostname = environ.get("REDIS_HOST", "localhost")
    port = environ.get("REDIS_PORT", 6379)
    password = environ.get("REDIS_PASSWORD", None)

    r = Inventa(hostname, port, password, SelfDescriptor.ServiceType, SelfDescriptor.ServiceId, InventaRole.Service, {})
    r.Start()
    return r

async def try_register_to_orchestrator(inventa: Inventa):
    try:
        await inventa.TryRegisterToOrchestrator(SignalingDescriptor.Encode(), 10, 3000)
    except Exception as e:
        print(f"Registration to signaling service was failed! Breaking down! {e}")
        raise e
    print(f"Registered to signaling service as {SelfDescriptor.Encode()}")

def exception_handler(loop, context):
    try:
        if context["future"]._coro and context["future"]._coro.__name__ == "try_register_to_orchestrator":
            loop.stop()
        else:
            raise context["exception"]
    except:
        traceback.print_exc()


def main():
    print("Welcome to Distributed Inference Pipeline - Inference Worker!")
    print("=================================")
    print("This module acts as Deep Learning Inference Worker.\n\n\n")

    start_http_server(8000)
    
    event_loop = asyncio.get_event_loop()
    event_loop.set_exception_handler(exception_handler)
    inventa = connect_to_redis()
    event_loop.create_task(try_register_to_orchestrator(inventa))
    
    for inference_task_cls in ExportedInferenceTasks:
        inference_task = inference_task_cls(inventa, SelfDescriptor)
        inference_task_metrics = create_metrics_for_task(inference_task.get_task_id())
        inference_task.set_metrics(inference_task_metrics)
        print("Starting inference task: ", inference_task.get_task_id(), " - ", inference_task.get_task_name())
        event_loop.create_task(inference_task.run())

    event_loop.run_forever()

if __name__ == '__main__':
    main()
