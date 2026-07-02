package webrtcservie

import (
	"sound-stage-backend/internal/config"

	"github.com/pion/webrtc/v4"
)

func NewPeerConnection(
	turn config.TurnConfig,
	onICECandidate func(webrtc.ICECandidateInit),
	onNegotiationNeeded func(webrtc.SessionDescription),
) (*webrtc.PeerConnection, error) {
	webrtcConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.relay.metered.ca:80"},
			},
		},
	}

	if turn.Username != "" && turn.Credential != "" {
		webrtcConfig.ICEServers = append(webrtcConfig.ICEServers,
			webrtc.ICEServer{
				URLs:       []string{"turn:global.relay.metered.ca:80?transport=tcp"},
				Username:   turn.Username,
				Credential: turn.Credential,
			},
			webrtc.ICEServer{
				URLs:       []string{"turns:global.relay.metered.ca:443?transport=tcp"},
				Username:   turn.Username,
				Credential: turn.Credential,
			},
		)
	}

	pc, err := webrtc.NewPeerConnection(webrtcConfig)
	if err != nil {
		return nil, err
	}

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			onICECandidate(c.ToJSON())
		}
	})

	pc.OnNegotiationNeeded(func() {
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			return
		}
		if err := pc.SetLocalDescription(offer); err != nil {
			return
		}
		onNegotiationNeeded(*pc.LocalDescription())
	})

	return pc, nil
}

func HandleOffer(pc *webrtc.PeerConnection, offer webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	if err := pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	return pc.LocalDescription(), nil
}

func HandleAnswer(pc *webrtc.PeerConnection, answer webrtc.SessionDescription) error {
	return pc.SetRemoteDescription(answer)
}

func AddICECandidate(pc *webrtc.PeerConnection, c webrtc.ICECandidateInit) error {
	return pc.AddICECandidate(c)
}

func NewForwardingTrack(remote *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error) {
	return webrtc.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability, "audio", "speaker")
}

func ForwardRTP(remote *webrtc.TrackRemote, local *webrtc.TrackLocalStaticRTP, stop <-chan struct{}) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, _, err := remote.Read(buf)
		if err != nil {
			return
		}
		if _, err := local.Write(buf[:n]); err != nil {
			return
		}
	}
}

func AddTrack(pc *webrtc.PeerConnection, trk *webrtc.TrackLocalStaticRTP) (*webrtc.RTPSender, error) {
	if pc == nil {
		return nil, nil
	}
	return pc.AddTrack(trk)
}

func RemoveTrack(pc *webrtc.PeerConnection, sender *webrtc.RTPSender) error {
	return pc.RemoveTrack(sender)
}
