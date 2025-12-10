package otp

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Entry represents an OTP with expiration
type Entry struct {
	Code      string
	ExpiresAt time.Time
}

// Service manages OTP generation, storage, and verification
type Service struct {
	store       map[string]Entry
	mu          sync.RWMutex
	expiry      time.Duration
	cleanupStop chan struct{}
	cleanupWg   sync.WaitGroup
}

// NewService initializes a new OTP service
func NewService(expiry time.Duration) OtpService {
	return &Service{
		store:       make(map[string]Entry),
		expiry:      expiry,
		cleanupStop: make(chan struct{}),
	}
}

// StartCleanup launches a background cleaner that removes expired OTPs
func (s *Service) StartCleanup(interval time.Duration) {
	s.cleanupWg.Add(1)
	go func() {
		defer s.cleanupWg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanupExpired()
			case <-s.cleanupStop:
				return
			}
		}
	}()
}

// StopCleanup stops the background cleaner gracefully
func (s *Service) StopCleanup() {
	close(s.cleanupStop)
	s.cleanupWg.Wait()
}

// Generate creates and stores an OTP for an identifier (e.g., email or phone)
func (s *Service) Generate(identifier string) string {
	otp := fmt.Sprintf("%04d", rand.Intn(10000)) // 4-digit OTP
	entry := Entry{
		Code:      otp,
		ExpiresAt: time.Now().Add(s.expiry),
	}

	s.mu.Lock()
	s.store[identifier] = entry
	s.mu.Unlock()

	return otp
}

// Verify checks if the provided OTP is valid
func (s *Service) Verify(identifier, input string) bool {
	s.mu.RLock()
	entry, exists := s.store[identifier]
	s.mu.RUnlock()

	if !exists || time.Now().After(entry.ExpiresAt) {
		return false
	}
	return input == entry.Code
}

// cleanupExpired removes expired OTPs from the store
func (s *Service) cleanupExpired() {
	now := time.Now()

	s.mu.Lock()
	for id, entry := range s.store {
		if now.After(entry.ExpiresAt) {
			delete(s.store, id)
		}
	}
	s.mu.Unlock()
}
