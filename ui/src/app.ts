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

import $ from "jquery";
import * as sdpTransform from "sdp-transform";

import "./style/main.css";

class Utils {
    static getBackendURL(relativePath: string) {
        return window.location.origin + "/" + relativePath;
    }

    static getBackendWSURL(relativePath: string) {
        return (
            "ws" +
            (window.location.protocol === "https:" ? "s" : "") +
            "://" +
            window.location.host +
            "/" +
            relativePath
        );
    }
}

class RTC {
    videoElm: any;
    localConnection: RTCPeerConnection;
    localTracks: MediaStreamTrack[] = [];

    constructor(videoElm: any) {
        this.videoElm = videoElm;
        this.localConnection = null;
    }

    createLocalPeerConnection() {
        const result = new RTCPeerConnection({
            iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        });
        result.onicecandidate = (e: RTCPeerConnectionIceEvent) => {
            if (e.candidate === null) {
                console.log('onicecandidate: localSessionDescription:\n', (<any>e.target).localDescription);
            } else {            
                console.log('New ICE candidate:', e.candidate);
            }
        };
        result.addEventListener("track", (e) => {
            console.log("onTrack", e);
        });
        result.onicecandidateerror = (e: Event) => {
            console.log("onicecandidateerror", "candidate address:", (<any>e).hostCandidate ?? '', "error text:", (<any>e).errorText ?? '', e);
        };
        result.onconnectionstatechange = (e: Event) => {
            this.updateStatus((<any>e.target).connectionState);
            console.log(
                "onconnectionstatechange",
                (<any>e.target).connectionState,
                "\n",
                e
            );
        };
        result.oniceconnectionstatechange = (e: Event) => {
            const peerConnection = <any>e.target;
            this.updateStatus(peerConnection.iceConnectionState);
            console.log(
                "oniceconnectionstatechange",
                peerConnection.iceConnectionState,
                "\n",
                e
            );
            if (peerConnection.iceConnectionState == "disconnected") {
                this.stop(true);
            } else if (peerConnection.iceConnectionState === 'connected' || peerConnection.iceConnectionState === 'completed') {
                // Get the stats for the peer connection
                peerConnection.getStats().then((stats: any) => {                
                    stats.forEach((report: any) => {
                        if (report.type === 'candidate-pair' && report.state === 'succeeded') {
                            const localCandidate = stats.get(report.localCandidateId);
                            const remoteCandidate = stats.get(report.remoteCandidateId);
                            console.log('Succeded Local Candidate:', report.localCandidateId, 'address:', localCandidate?.address, 'object:', localCandidate);
                            console.log('Succeded Remote Candidate:', report.remoteCandidateId, 'address:', remoteCandidate?.address, 'object:', remoteCandidate);
                        }
                    });
                });
            }
        };
        result.onicegatheringstatechange = (e: Event) => {
            console.log(
                "onicegatheringstatechange",
                (<any>e.target).iceGatheringState,
                "\n",
                e
            );
        };
        result.onnegotiationneeded = (e: Event) => {
            console.log("onnegotiationneeded", e);
        };
        result.onsignalingstatechange = (e: Event) => {
            console.log(
                "onsignalingstatechange",
                (<any>e.target).signalingState,
                "\n",
                e
            );
        };
        return result;
    }

    createLocalTracks(): Promise<MediaStream> {
        if (!navigator.mediaDevices) {
            throw new Error("You are trying to access a remote web app using HTTP. Browsers does not allow using WebRTC while connecting not secure HTTP. Please try connecting the web app with HTTPS instead.");
        }
        return navigator.mediaDevices.getUserMedia({
            video: {
                height: 720,
                frameRate: { max: 30 },
            },
            audio: false,
        });
    }

    start() {
        this.updateStatus("starting...");
        this.localConnection = this.createLocalPeerConnection();
        return this.createLocalTracks()
            .then((stream) => {
                stream.getTracks().forEach((track) => {
                    this.localTracks.push(track);
                    this.localConnection.addTrack(track);
                    if (track.kind == "video") {
                        this.videoElm.srcObject = new MediaStream([track]);
                        this.videoElm.play();
                    }
                });
            })
            .catch((e) => {
                console.error("Error while starting: ", e);
                alert("Error while starting:\n" + e);
                this.stop(true);
            })
            .then(() => signaling.connect());
    }

    stop(closeConnection: Boolean) {
        if (this.videoElm.srcObject) {
            try {
                this.videoElm.stop();
                this.videoElm.srcObject = null;
            } catch {
                // Do nothing
            }
            signaling.resetCanvas();
        }
        this.localTracks.forEach((localTrack) => {
            localTrack.enabled = false;
            localTrack.stop();
        });
        if (closeConnection && this.localConnection) {
            const updateStatusRequired = !(this.localConnection.iceConnectionState == "disconnected");
            this.localConnection.close();
            this.localConnection = null;
            if (updateStatusRequired) {
                this.updateStatus("closed");
            }
        }
        this.localTracks = [];
        console.log("Stopping tracks. closeConnection: ", closeConnection);
    }

    sendSdpToSignaling(parsedSdp: sdpTransform.SessionDescription) {
        console.log("sendSdpToSignaling", parsedSdp);
        signaling.ws.send(
            JSON.stringify({ type: "SdpOfferAnswer", data: {"sdp": sdpTransform.write(parsedSdp)} })
        );
    }

    acceptOffer(offerSdp: string) {
        return this.localConnection
            .setRemoteDescription({
                type: "offer",
                sdp: offerSdp,
            })
            .then(() => {
                return this.localConnection
                    .createAnswer()
                    .then((answer: RTCSessionDescriptionInit) => {
                        console.log("answer", answer.type, answer.sdp);
                        const parsedSdp: sdpTransform.SessionDescription = sdpTransform.parse(
                            answer.sdp
                        );
                        this.sendSdpToSignaling(parsedSdp);
                        this.localConnection.setLocalDescription(answer);
                    });
            })
            .catch((e) => {
                console.error("Error while acceptOffer: ", e);
                alert("Error while acceptOffer:\n" + e);
                this.stop(true);
            });
    }

    updateStatus(connStatus: string) {
        $("#LblConnStatus").text("Connection status: " + connStatus);
    }
}

class Signaling {
    ws: WebSocket;
    canvasCtx: CanvasRenderingContext2D;

    connect() {
        const signalingUrl = Utils.getBackendWSURL("signaling");
        console.log(`Start connect() ${signalingUrl}`);
        this.ws = new WebSocket(signalingUrl);

        this.ws.onopen = () => {
            console.log("client side socket connection established");
        };

        this.ws.onclose = () => {
            console.log("client side socket connection disconnected");
            rtc.stop(true);
        };

        this.ws.onerror = (error) => {
            console.log("Websocket error:", error);
            rtc.stop(true);
            alert(
                "Could not connect to websocket " + (<WebSocket>error.target).url + ". Ready state: " +
                (<WebSocket>error.target).readyState
            );
        };

        this.ws.onmessage = (message) => {
            let data = null;
            try {
                data = message.data ? JSON.parse(message.data) : null;
            } catch {
                // Do nothing
            }
            console.log("Received from WS:", message.data);
            if (!data) {
                return;
            }
            switch (data.type) {
                case "Welcome":
                    signaling.ws.send(
                        JSON.stringify({
                            type: "Join",
                            data: {
                                tenantId: "defaultTenant",
                            },
                        })
                    );

                    break;
                case "JoinError":
                    console.error("Error while starting: ", data.data);
                    rtc.stop(true);
                    alert("Error while joining:\n" + data.data);
                    break;
                case "SdpOffer":
                   const offer = JSON.parse(data.data);
                    try {
                        console.log("offerSdp",  offer.sdp);
                        rtc.acceptOffer(offer.sdp);
                    } catch (e) {
                        console.error(e);
                        rtc.stop(true);
                        alert(e);
                    }
                    break;
                case "Prediction":
                    this.processPredictions(data.data);
                    break;
            }
        };
    }

    close() {
        this.ws.close();
    }

    resetCanvas() {
        const canvasElm = document.getElementById("canvasElm") as HTMLCanvasElement;
        const videoElm = document.getElementById("videoElm") as HTMLCanvasElement;
        var ctx = canvasElm.getContext("2d");
        signaling.canvasCtx = ctx;
        canvasElm.width = ($(videoElm).width());
        canvasElm.height = ($(videoElm).height());
        console.log("Video was resized: ",canvasElm.width, "x", canvasElm.height);
        signaling.canvasCtx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);
    }

    processPredictions(predictionsData: any) {
        const pcount = parseInt(predictionsData.pcount);
        signaling.canvasCtx.clearRect(0, 0, signaling.canvasCtx.canvas.width, signaling.canvasCtx.canvas.height);
        if (pcount == 0) {
            return;
        }
        signaling.canvasCtx.font = "20px sans-serif";

        const resolutionH = parseInt(predictionsData.res);
        const scale = this.canvasCtx.canvas.height / resolutionH;

        for (let i = 0; i < pcount; i++) {
            const predictionParts = predictionsData["p" + i].split("|");
            const textLines = [predictionParts[0], parseFloat(predictionParts[1])];
            const boxStr = predictionParts[2].split(";");
            const boxNum = boxStr.map((item: string) => parseInt(item) * scale);
            signaling.canvasCtx.beginPath();
            const boxX = boxNum[0];
            const boxY = boxNum[1];
            const boxW = boxNum[2] - boxX;
            const boxH = boxNum[3] - boxY;
            signaling.canvasCtx.rect(
                boxX, 
                boxY, 
                boxW, 
                boxH);
            signaling.canvasCtx.stroke();
            const metrics = signaling.canvasCtx.measureText("A");
            const fontHeight = (metrics.fontBoundingBoxAscent || metrics.actualBoundingBoxAscent * 1.5) + (metrics.fontBoundingBoxDescent || metrics.actualBoundingBoxDescent * 1.5);

            for (let i = 0; i < textLines.length; i++) {
                signaling.canvasCtx.fillText(textLines[i], boxX, boxY + ((i + 1) * fontHeight));
            }    
        }
    }
}

const rtc = new RTC(document.getElementById("videoElm"));
(<any>window).rtc = rtc;

const signaling = new Signaling();
(<any>window).signaling = signaling;

function initApp() {
    $("#BtnCreatePC").on("click", () => rtc.start());
    $("#BtnStopPC").on("click", () => rtc.stop(true));
    $(window).bind("beforeunload", () => rtc.stop(true));
    
    const videoElm = document.getElementById("videoElm") as HTMLCanvasElement;

    videoElm.addEventListener("resize", () => {
        setTimeout(() => {
            signaling.resetCanvas();
        }, 0);
    });
}

initApp();
