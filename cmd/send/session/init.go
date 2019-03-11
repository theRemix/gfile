package session

import (
	"fmt"

	"github.com/antonito/gfile/pkg/utils"
	"github.com/pions/webrtc"
)

// Connect starts a connection and waits till it ends
func (s *Session) Connect() error {
	if err := s.createConnection(); err != nil {
		return err
	}
	if err := s.createDataChannel(); err != nil {
		return err
	}

	sdpChan := utils.HTTPSDPServer()

	if err := s.createOffer(); err != nil {
		return err
	}

	// Wait for the answer to be pasted
	fmt.Println(`Please, provide the SDP via:
echo "$SDP" | gfile sdp`)
	answer := webrtc.SessionDescription{}
	for {
		if err := utils.Decode(<-sdpChan, &answer); err == nil {
			break
		}
		fmt.Println("Invalid SDP, try aagain...")
	}

	// Apply the answer as the remote description
	if err := s.peerConnection.SetRemoteDescription(answer); err != nil {
		return err
	}

	<-s.done
	return nil
}

func (s *Session) createConnection() error {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}
	s.peerConnection = peerConnection
	s.setStateManager()

	return nil
}

func (s *Session) createOffer() error {
	// Create an offer to send to the browser
	offer, err := s.peerConnection.CreateOffer(nil)
	if err != nil {
		return err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	if err := s.peerConnection.SetLocalDescription(offer); err != nil {
		return err
	}
	offer.SDP = utils.StripSDP(offer.SDP)

	// Output the offer in base64 so we can paste it in browser
	encoded, err := utils.Encode(offer)
	if err != nil {
		return err
	}
	fmt.Fprintln(s.sdpOutput, encoded)
	return nil
}

func (s *Session) createDataChannel() error {
	//ordered := true
	//maxPacketLifeTime := uint16(0xFFFF)
	dataChannel, err := s.peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		return err
	}
	go s.readFile()
	s.dataChannel = dataChannel
	s.dataChannel.OnOpen(s.onOpenHandler())
	s.dataChannel.OnClose(s.onCloseHandler())
	return nil
}
