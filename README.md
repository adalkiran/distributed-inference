# **Distributed Deep Learning Inference Pipeline**

[![LinkedIn](https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white&style=flat-square)](https://www.linkedin.com/in/alper-dalkiran/)
[![Twitter](https://img.shields.io/badge/Twitter-1DA1F2?style=for-the-badge&logo=twitter&logoColor=white&style=flat-square)](https://twitter.com/aalperdalkiran)
![HitCount](https://hits.dwyl.com/adalkiran/distributed-inference.svg?style=flat-square)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

Cross-language and distributed deep learning inference pipeline for WebRTC video streams over Redis Streams. Currently supports YOLOX model, which can run well on CPU.

This project consists of WebRTC signaling and orchestrator service(Go), WebRTC media server service (Go), YOLOX model deep learning inference service (Python), and Web front-end (TypeScript).

Uses
* [Inventa for Go](https://github.com/adalkiran/go-inventa) and [Inventa for Python](https://github.com/adalkiran/py-inventa) to do service orchestration and to make RPC over Redis. For more information, you can check out [Inventa Examples](https://github.com/adalkiran/inventa-examples) repo.
* [Pion WebRTC](https://github.com/pion/webrtc) library to implement WebRTC related stuff.
* [YOLOX model](https://github.com/Megvii-BaseDetection/YOLOX) for object detection from captured images.

<br>

## **WHY THIS PROJECT?**

This project aims to demonstrate an approach to designing cross-language and distributed pipeline in deep learning/machine learning domain. Tons of demos and examples can be found on the internet which are developed end to end only (mostly) in Python. But this project is one of cross-language examples.

<br>

## **INSTALLATION and RUNNING**

This project was designed to run in Docker Container. For some configurations, you can check out docker-compose.yaml and .env file in the root folder.

Docker Compose file creates some containers, with some replica instances:
* **redis:** Runs a Redis instance.
* **web:** Runs an Nginx instance for proxy passing HTTP and WebSocket endpoints.
* **signaling service:** The only orchestrator in the project. Other services will register themselves to this application. Also, it serves a WebSocket for doing WebRTC signaling function. When a WebRTC comes to join, this service selects one of registered media bridge services and brings the WebRTC client and media bridge together. Written in Go language.

* **mediabridge service:** The service will register itself to the orchestrator, and can respond to "sdp-offer-req" and "sdp-accept-offer-answer" procedure calls. Also uses [Pion WebRTC](https://github.com/pion/webrtc) library to serve as a WebRTC server on a UDP port. Written in Go language.
    <br>
    Designed to run as more than one instances, but currently can run only one instance because it should expose a UDP port from Docker container, and different container replicas on same host should be assigned different and available port numbers and the application must know the exposed host port number, in this stage, it couldn't be achieved to dynamically manage this. Maybe it can be achieved on Kubernetes.

* **inference:** The service will register itself to the orchestrator, and can keep track of "images" Redis Stream which streams JPEG image data of video frames. Written in Python language. It makes inferences on incoming images with YOLOX model for object detection.
    <br>
    Can be more than one, by docker-compose.yml file's replica values.

* **ui:** The web frontend. It gets a media stream from webcam and forwards it to assigned mediabridge service via WebRTC.

You can run it in production mode or development mode.

### **Production Mode**

* Clone this repo and run in terminal:

```sh
$ docker-compose up -d
```

* Wait until Go and Python modules were installed and configured. This can take some time. You can check out the download status by:

```sh
$ docker-compose logs -f
```

* After waiting for enough time, open a web browser and visit http://localhost:9000 (Tested on Chrome)

### <a name="dev-mode"></a>**Development Mode: VS Code Remote - Containers**

To continue with VS Code and if this is your first time to work with Remote Containers in VS Code, you can check out [this link](https://code.visualstudio.com/docs/remote/containers) to learn how Remote Containers work in VS Code and follow the installation steps of [Remote Development extension pack](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.vscode-remote-extensionpack).

Then, follow these steps:

* Clone this repo to your local filesystem
* Open the folder "distributed-inference" with VS Code by "Open Folder..." command. This opens the root folder of the project.
* Press <kbd>F1</kbd> and select **"Remote Containers: Open Folder in Container..."** then select one of folders in the root folder, not the root folder itself. You can select any of the services, which you want to develop.
* This command creates (if they don't exist) required containers in Docker, then connects inside of distributed-inference-[your-selected-service] container for development and debugging purposes.
* Wait until the containers are created, configured, and related VS Code server extensions installed inside the container. This can take some time. VS Code can ask for some required installations, click "Install All" for these prompts.
* After completion of all installations, press <kbd>F5</kbd> to start server application.
    <br>
    **Note:** Maybe you must kill existing running service processes by terminal.
* Then, you can keep track of other services with docker logs.
* Then, open a web browser and visit http://localhost:9000 (Tested on Chrome)

<br>

## **LICENSE**

Distributed Deep Learning Inference Pipeline project is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.