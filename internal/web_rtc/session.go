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
}

func (s *Session) SetLocalTrack(track *pion.TrackLocalStaticRTP) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.localTrack = track
}

func (s *Session) LocalTrack() *pion.TrackLocalStaticRTP {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.localTrack
}

func (s *Session) Stop() <-chan struct{} {
	return s.stop
}

func (s *Session) close() error {
	close(s.stop)
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
	session := &Session{PC: pc, stop: make(chan struct{})}

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
