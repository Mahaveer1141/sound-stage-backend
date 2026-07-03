package webrtc

import (
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/pkg/httpx"

	pion "github.com/pion/webrtc/v4"
)

func NewPeerConnection(
	cfg *config.Config,
	onICECandidate func(pion.ICECandidateInit),
	onNegotiationNeeded func(pion.SessionDescription),
) (*pion.PeerConnection, error) {
	webrtcConfig := pion.Configuration{ICEServers: buildIceServers(cfg)}

	pc, err := pion.NewPeerConnection(webrtcConfig)
	if err != nil {
		return nil, err
	}

	pc.OnICECandidate(func(c *pion.ICECandidate) {
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

func HandleOffer(pc *pion.PeerConnection, offer pion.SessionDescription) (*pion.SessionDescription, error) {
	if pc == nil {
		return nil, httpx.ErrPeerConnectionNotFound
	}
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

func HandleAnswer(pc *pion.PeerConnection, answer pion.SessionDescription) error {
	if pc == nil {
		return httpx.ErrPeerConnectionNotFound
	}
	return pc.SetRemoteDescription(answer)
}

func AddICECandidate(pc *pion.PeerConnection, c pion.ICECandidateInit) error {
	if pc == nil {
		return httpx.ErrPeerConnectionNotFound
	}
	return pc.AddICECandidate(c)
}

func NewForwardingTrack(remote *pion.TrackRemote) (*pion.TrackLocalStaticRTP, error) {
	return pion.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability, "audio", "speaker")
}

func ForwardRTP(remote *pion.TrackRemote, local *pion.TrackLocalStaticRTP, stop <-chan struct{}) {
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

func AddTrack(pc *pion.PeerConnection, trk *pion.TrackLocalStaticRTP) (*pion.RTPSender, error) {
	if pc == nil {
		return nil, httpx.ErrPeerConnectionNotFound
	}
	return pc.AddTrack(trk)
}

func RemoveTrack(pc *pion.PeerConnection, sender *pion.RTPSender) error {
	if pc == nil {
		return httpx.ErrPeerConnectionNotFound
	}
	return pc.RemoveTrack(sender)
}

func buildIceServers(cfg *config.Config) []pion.ICEServer {
	iceServers := []pion.ICEServer{
		{
			URLs: []string{cfg.WebRTC.StunURL},
		},
	}
	if cfg.Server.Environment != "development" && cfg.WebRTC.TurnUsername != "" &&
		cfg.WebRTC.TurnCredential != "" && cfg.WebRTC.TurnURL != "" {
		iceServers = append(iceServers,
			pion.ICEServer{
				URLs:       []string{"turn:" + cfg.WebRTC.TurnURL + ":80?transport=tcp"},
				Username:   cfg.WebRTC.TurnUsername,
				Credential: cfg.WebRTC.TurnCredential,
			},
			pion.ICEServer{
				URLs:       []string{"turns:" + cfg.WebRTC.TurnURL + ":443?transport=tcp"},
				Username:   cfg.WebRTC.TurnUsername,
				Credential: cfg.WebRTC.TurnCredential,
			},
		)
	}
	return iceServers
}
