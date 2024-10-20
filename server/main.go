package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
	"github.com/quic-go/quic-go/http3"
)

type SignalMessage struct {
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
}

var peerConnection *webrtc.PeerConnection

func handleSignaling(w http.ResponseWriter, r *http.Request) {
	// CORS 헤더 추가
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered from panic: %v", err)
		}
	}()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var signal SignalMessage
	err = json.Unmarshal(body, &signal)
	if err != nil {
		http.Error(w, "Failed to unmarshal JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received Signal: %+v\n", signal)

	if signal.SDP != nil {
		// 피어 연결 설정
		peerConnection, err = webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			log.Println("Error creating PeerConnection:", err)
			http.Error(w, "Error creating PeerConnection", http.StatusInternalServerError)
			return
		}

		// ICE 후보 처리 핸들러 설정
		peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate == nil {
				return
			}
			iceCandidate := candidate.ToJSON()                  // 값 복사
			response := SignalMessage{Candidate: &iceCandidate} // 복사된 값을 포인터로 전달
			responseData, err := json.Marshal(response)
			if err != nil {
				log.Println("Failed to marshal ICE candidate:", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(responseData)
		})

		err = peerConnection.SetRemoteDescription(*signal.SDP)
		if err != nil {
			log.Println("Error setting remote description:", err)
			http.Error(w, "Error setting remote description", http.StatusInternalServerError)
			return
		}

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			log.Println("Error creating answer:", err)
			http.Error(w, "Error creating answer", http.StatusInternalServerError)
			return
		}

		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			log.Println("Error setting local description:", err)
			http.Error(w, "Error setting local description", http.StatusInternalServerError)
			return
		}

		response := SignalMessage{SDP: &answer}
		responseData, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
	}

	if signal.Candidate != nil && peerConnection != nil {
		err := peerConnection.AddICECandidate(*signal.Candidate)
		if err != nil {
			log.Println("Error adding ICE candidate:", err)
			http.Error(w, "Error adding ICE candidate", http.StatusInternalServerError)
		}
	}
}

func main() {
	// 정적 파일 제공 (HTML 파일 및 관련 리소스 제공)
	fileServer := http.FileServer(http.Dir("./static"))

	// HTTP 라우팅 설정
	mux := http.NewServeMux()
	mux.Handle("/", fileServer)
	mux.HandleFunc("/signaling", handleSignaling)

	// HTTP/3 서버 설정 (QUIC 기반)
	h3Server := &http3.Server{
		Addr:    ":443",
		Handler: mux,
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Recovered from panic in HTTP/3 server: %v", err)
			}
		}()

		log.Println("Starting HTTP/3 signaling server on :443")
		err := h3Server.ListenAndServeTLS("server-cert.pem", "server-key.pem")
		if err != nil {
			log.Fatal("Failed to start HTTP/3 server:", err)
		}
	}()

	select {}
}
