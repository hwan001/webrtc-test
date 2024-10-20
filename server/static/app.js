// app.js
const videoElement = document.getElementById('remoteVideo');

// WebRTC 연결 생성
const peerConnection = new RTCPeerConnection({
    iceServers: [{ urls: "stun:stun.l.google.com:19302" }] // 기본적인 STUN 서버 설정
});

// 트랙 수신 시 비디오 요소에 스트림 연결
peerConnection.ontrack = (event) => {
    console.log('Received remote track');
    videoElement.srcObject = event.streams[0];
};

// ICE 후보 수신 처리
peerConnection.onicecandidate = (event) => {
    if (event.candidate) {
        console.log('Sending ICE candidate');
        fetch('https://localhost/signaling', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ candidate: event.candidate })
        });
    }
};

// 시그널링 서버에 연결하고 Offer 전송
async function startWebRTC() {
    // SDP Offer 생성
    const offer = await peerConnection.createOffer();
    await peerConnection.setLocalDescription(offer);

    // HTTP/3 서버에 Offer 전송
    const response = await fetch('/signaling', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ sdp: offer })
    });

    const responseData = await response.json();
    const remoteSDP = new RTCSessionDescription(responseData.sdp);

    // Remote SDP 설정
    await peerConnection.setRemoteDescription(remoteSDP);
}

// WebRTC 연결 시작
startWebRTC().catch(console.error);