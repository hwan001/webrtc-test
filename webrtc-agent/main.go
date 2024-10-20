package main

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var peerConnection *webrtc.PeerConnection

func main() {
	// Connect to the signaling server
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/signal"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to signaling server: %v", err)
	}
	defer conn.Close()

	// WebRTC configuration
	config := webrtc.Configuration{}
	peerConnection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create PeerConnection: %v", err)
	}

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion")
	if err != nil {
		log.Fatalf("Failed to create video track: %v", err)
	}

	// Add the video track to the connection
	if _, err := peerConnection.AddTrack(videoTrack); err != nil {
		log.Fatalf("Failed to add track: %v", err)
	}

	// Set up handlers to receive signaling messages from the signaling server
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				return
			}

			// Handle incoming messages (SDP and ICE candidates)
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Failed to parse message: %v", err)
				continue
			}

			if sdp, ok := msg["sdp"]; ok {
				// If SDP offer/answer is received
				offer := webrtc.SessionDescription{}
				sdpBytes, _ := json.Marshal(sdp)
				if err := json.Unmarshal(sdpBytes, &offer); err != nil {
					log.Printf("Failed to parse SDP: %v", err)
					continue
				}
				peerConnection.SetRemoteDescription(offer)

				// If received offer, create an answer
				if offer.Type == webrtc.SDPTypeOffer {
					answer, err := peerConnection.CreateAnswer(nil)
					if err != nil {
						log.Printf("Failed to create answer: %v", err)
						continue
					}
					peerConnection.SetLocalDescription(answer)

					answerMsg, _ := json.Marshal(map[string]interface{}{
						"sdp": answer,
					})
					conn.WriteMessage(websocket.TextMessage, answerMsg)
				}
			}
		}
	}()

	// Main routine to keep the program running
	select {}
}
