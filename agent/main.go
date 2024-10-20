package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/quic-go/quic-go/http3"
)

type SignalMessage struct {
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
}

func main() {
	// WebRTC 피어 연결 설정
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Fatal("Error creating WebRTC peer connection:", err)
	}

	// 화면 캡처 트랙 생성
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType: "video/vp8",
	}, "video", "pion")
	if err != nil {
		log.Fatal("Error creating video track:", err)
	}
	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Fatal("Error adding video track:", err)
	}

	// HTTP/3 클라이언트 설정
	httpClient := &http.Client{
		Transport: &http3.RoundTripper{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 자체 서명된 인증서 검증 비활성화
		},
	}

	// SDP 생성 및 전송
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Fatal("Error creating offer:", err)
	}
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Fatal("Error setting local description:", err)
	}

	signal := SignalMessage{SDP: &offer}
	jsonData, err := json.Marshal(signal)
	if err != nil {
		log.Fatal("Error marshalling signal message:", err)
	}

	// HTTP/3 POST 요청으로 시그널링 메시지 전송 및 응답 처리
	resp, err := httpClient.Post("https://localhost:443/signaling", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal("Error sending signal message:", err)
	}
	defer resp.Body.Close()

	// 응답 수신 및 처리
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading response:", err)
	}

	var responseSignal SignalMessage
	err = json.Unmarshal(responseBody, &responseSignal)
	if err != nil {
		log.Fatal("Error unmarshalling response signal:", err)
	}

	if responseSignal.SDP != nil {
		err = peerConnection.SetRemoteDescription(*responseSignal.SDP)
		if err != nil {
			log.Fatal("Error setting remote description:", err)
		}
	}

	// 제어 메시지 수신 및 처리
	dataChannel, err := peerConnection.CreateDataChannel("control", nil)
	if err != nil {
		log.Fatal("Error creating WebRTC data channel:", err)
	}

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Control Message Received: %s\n", string(msg.Data))
	})

	// ffmpeg 명령어 실행 (avfoundation 사용)
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-i", "1", "-vcodec", "libvpx", "-f", "webm", "-")
	ffmpegOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("Error creating ffmpeg stdout pipe:", err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal("Error starting screen capture:", err)
	}

	// WebRTC 연결 종료 시 화면 캡처 중지
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection state changed: %s\n", state.String())

		if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
			log.Println("Connection closed or failed, stopping screen capture.")
			cmd.Process.Kill()
		}
	})

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE connection state changed: %s\n", state.String())
	})

	// 화면 캡처 데이터를 WebRTC 트랙에 전송
	packetizer := rtp.NewPacketizer(
		1400,                     // Maximum RTP packet size
		96,                       // Payload type for VP8
		12345,                    // SSRC
		&codecs.VP8Payloader{},   // Payload handler
		rtp.NewFixedSequencer(1), // Sequence number generator
		90000,                    // Clock rate
	)

	buf := make([]byte, 1400)
	for {
		n, err := ffmpegOut.Read(buf)
		if err != nil {
			log.Println("Error reading ffmpeg output:", err)
			break
		}

		timestamp := uint32(time.Now().UnixNano() / int64(time.Millisecond)) // RTP 타임스탬프 변환
		packets := packetizer.Packetize(buf[:n], timestamp)
		for _, packet := range packets {
			err = videoTrack.WriteRTP(packet)
			if err != nil {
				log.Println("Error writing to video track:", err) // 트랙 쓰기 실패 시 에러 로그 추가
				break
			}
		}
	}
}
