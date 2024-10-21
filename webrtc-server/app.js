const signalingServerUrl = "wss://h001.666lab.org/signal";
const configuration = {
  iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
};
const peerConnection = new RTCPeerConnection(configuration);

// Establish WebSocket connection for signaling
const signalingSocket = new WebSocket(signalingServerUrl);

// Handle WebSocket messages
signalingSocket.onmessage = async (message) => {
  const data = JSON.parse(message.data);

  if (data.sdp) {
    await peerConnection.setRemoteDescription(new RTCSessionDescription(data.sdp));

    if (data.sdp.type === "offer") {
      const answer = await peerConnection.createAnswer();
      await peerConnection.setLocalDescription(answer);
      signalingSocket.send(JSON.stringify({ sdp: answer }));
    }
  } else if (data.candidate) {
    await peerConnection.addIceCandidate(new RTCIceCandidate(data.candidate));
  }
};


peerConnection.ontrack = (event) => {
  console.log('Received remote track:', event);
  const videoElement = document.getElementById("remoteVideo");
  videoElement.srcObject = event.streams[0];
  console.log('Video stream set to video element');
};

peerConnection.oniceconnectionstatechange = () => {
  console.log('ICE connection state changed:', peerConnection.iceConnectionState);
};

// Create an offer
async function startConnection() {
  const offer = await peerConnection.createOffer();
  await peerConnection.setLocalDescription(offer);
  signalingSocket.send(JSON.stringify({ sdp: offer }));
}