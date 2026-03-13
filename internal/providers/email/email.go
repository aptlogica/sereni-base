// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"serenibase/internal/config"
	"sync"
	"time"
)

// Service handles sending emails via HTTP request with a worker queue
type Service struct {
	url        string
	queue      chan EmailJob
	workerWg   sync.WaitGroup
	workerStop chan struct{}
}

// NewService initializes a new email service
func NewService(cfg config.EmailConfig, queueSize int, emailTemplateService EmailTemplateService) EmailService {
	return &Service{
		url:        cfg.URL,
		queue:      make(chan EmailJob, queueSize),
		workerStop: make(chan struct{}),
	}
}

// Start launches N email workers
func (s *Service) Start(workers int) {
	for i := 0; i < workers; i++ {
		s.workerWg.Add(1)
		go s.worker(i + 1)
	}
}

// Stop gracefully shuts down workers
func (s *Service) Stop() {
	close(s.workerStop) // signal workers to stop
	close(s.queue)      // unblock workers waiting on queue
	s.workerWg.Wait()   // wait for workers to finish
}

// Enqueue adds a new email job to the queue
func (s *Service) Enqueue(job EmailJob) {
	select {
	case s.queue <- job:
	default:
		log.Printf("Email queue full, dropping email to %s", job.To)
	}
}

// worker processes email jobs
func (s *Service) worker(id int) {
	defer s.workerWg.Done()
	for {
		select {
		case <-s.workerStop:
			return
		case job, ok := <-s.queue:
			if !ok {
				return
			}
			if err := s.sendEmail(job); err != nil {
				fmt.Printf("Worker %d: failed to send email to %s: %v\n", id, job.To, err)
				log.Printf("Worker %d: failed to send email to %s: %v", id, job.To, err)
			} else {
				log.Printf("Worker %d: email sent to %s", id, job.To)
			}
		}
	}
}

// sendEmail sends the email using HTTP POST
func (s *Service) sendEmail(job EmailJob) error {
	payload := map[string]interface{}{
		"to":      []string{job.To},
		"subject": job.Subject,
		"body":    job.Body,
		"is_html": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/send", s.url),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("email service returned non-success status: %d", resp.StatusCode)
	}

	return nil
}
