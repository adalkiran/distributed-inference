ARG VARIANT=3.9.13-slim-bullseye
FROM python:${VARIANT}

COPY entrypoint.sh entrypoint-dev.sh requirements_cpu.txt /

WORKDIR /workspace

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install curl cmake build-essential protobuf-compiler libprotoc-dev ffmpeg libsm6 libxext6 libjpeg-dev libpng-dev \
    && chown -R 42420:42420 /workspace \
    && python -m pip install --upgrade pip \
    && pip install --upgrade wheel setuptools \
    && pip install torch \
    && pip install -r /requirements_cpu.txt \
    && curl -L -o /home/yolox_nano.onnx https://github.com/Megvii-BaseDetection/YOLOX/releases/download/0.1.1rc0/yolox_nano.onnx \
    && chmod +x /entrypoint*.sh

ENTRYPOINT "/entrypoint.sh"