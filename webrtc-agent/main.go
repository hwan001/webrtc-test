package main

import (
	"encoding/json"
	"io"
	"log"
	"net/url"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

var peerConnection *webrtc.PeerConnection

func main() {
	// Connect to the signaling server
	u := url.URL{Scheme: "wss", Host: "h001.666lab.org", Path: "/signal"}
	log.Printf("Connecting to signaling server at %s...", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to signaling server: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to signaling server.")

	// WebRTC configuration
	config := webrtc.Configuration{}
	peerConnection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create PeerConnection: %v", err)
	}
	log.Println("PeerConnection created.")

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/vp8"}, "video", "pion")
	if err != nil {
		log.Fatalf("Failed to create video track: %v", err)
	}
	log.Println("Video track created.")

	// Add the video track to the connection
	if _, err := peerConnection.AddTrack(videoTrack); err != nil {
		log.Fatalf("Failed to add track: %v", err)
	}
	log.Println("Video track added to PeerConnection.")

	// Create and send SDP Offer to the signaling server
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Fatalf("Failed to create offer: %v", err)
	}
	if err := peerConnection.SetLocalDescription(offer); err != nil {
		log.Fatalf("Failed to set local description: %v", err)
	}

	offerMsg, err := json.Marshal(map[string]interface{}{
		"sdp": offer,
	})
	if err != nil {
		log.Fatalf("Failed to marshal offer: %v", err)
	}

	// Send the SDP Offer to the signaling server
	if err := conn.WriteMessage(websocket.TextMessage, offerMsg); err != nil {
		log.Fatalf("Failed to send offer to signaling server: %v", err)
	}
	log.Println("SDP offer sent to signaling server.")

	// Start capturing video from the screen using FFmpeg
	go func() {
		cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-i", "3", "-vcodec", "libvpx", "-f", "webm", "-")
		ffmpegOut, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal("Error creating ffmpeg stdout pipe:", err)
		}

		err = cmd.Start()
		if err != nil {
			log.Fatal("Error starting ffmpeg:", err)
		}

		buf := make([]byte, 4096)
		for {
			n, err := ffmpegOut.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Error reading ffmpeg output: %v", err)
				break
			}

			// Write the captured frame data to the video track
			err = videoTrack.WriteSample(media.Sample{Data: buf[:n], Duration: time.Second / 30})
			if err != nil {
				log.Printf("Failed to write sample to video track: %v", err)
				break
			}
			log.Println("Sent a video frame")
		}
	}()

	// Set up handlers to receive signaling messages from the signaling server
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Failed to read message from signaling server: %v", err)
				return
			}
			log.Println("Received signaling message from server.")

			// Handle incoming messages (SDP and ICE candidates)
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Failed to parse signaling message: %v", err)
				continue
			}

			if sdp, ok := msg["sdp"]; ok {
				// If SDP answer is received
				answer := webrtc.SessionDescription{}
				sdpBytes, _ := json.Marshal(sdp)
				if err := json.Unmarshal(sdpBytes, &answer); err != nil {
					log.Printf("Failed to parse SDP: %v", err)
					continue
				}
				log.Printf("Received SDP message of type: %s", answer.Type)

				if err := peerConnection.SetRemoteDescription(answer); err != nil {
					log.Printf("Failed to set remote description: %v", err)
					continue
				}
			}
		}
	}()

	go func() {
		for {
			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}
			time.Sleep(30 * time.Second) // Send ping every 30 seconds
		}
	}()

	// Main routine to keep the program running
	log.Println("Agent is running and ready to handle messages.")
	select {}
}