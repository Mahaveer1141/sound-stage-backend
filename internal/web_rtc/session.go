package webrtc

import (
	"sync"

	pion "github.com/pion/webrtc/v4"
)

type Session struct {
	PC *pion.PeerConnection

	mu         sync.RWMutex
	localTrack *pion.TrackLocalStaticRTP
	stop       chan struct{}
	senders    map[string]*pion.RTPSender
}

func (s *Session) StartPublishing(track *pion.TrackLocalStaticRTP) <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.localTrack = track
	s.stop = make(chan struct{})

	return s.stop
}

func (s *Session) StopPublishing() map[string]*pion.RTPSender {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}
	s.localTrack = nil

	senders := s.senders
	s.senders = make(map[string]*pion.RTPSender)

	return senders
}

func (s *Session) LocalTrack() *pion.TrackLocalStaticRTP {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.localTrack
}

func (s *Session) AddSender(clientID string, sender *pion.RTPSender) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.senders[clientID] = sender
}

func (s *Session) close() error {
	s.StopPublishing()
	return s.PC.Close()
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

func (s *SessionStore) Add(clientID string, pc *pion.PeerConnection) *Session {
	session := &Session{PC: pc, senders: make(map[string]*pion.RTPSender)}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[clientID] = session

	return session
}

func (s *SessionStore) Get(clientID string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[clientID]
}

func (s *SessionStore) Remove(clientID string) error {
	s.mu.Lock()
	session, ok := s.sessions[clientID]
	delete(s.sessions, clientID)
	s.mu.Unlock()

	if !ok {
		return nil
	}

	return session.close()
}
