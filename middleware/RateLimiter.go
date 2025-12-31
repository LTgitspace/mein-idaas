package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// BannedIP tracks banned IPs and their expiration
type BannedIP struct {
	BannedUntil time.Time
}

// IPBanStorage implements custom storage for rate limiter with ban functionality
type IPBanStorage struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	bans     map[string]*BannedIP
}

// NewIPBanStorage creates a new IP ban storage
func NewIPBanStorage() *IPBanStorage {
	storage := &IPBanStorage{
		requests: make(map[string][]time.Time),
		bans:     make(map[string]*BannedIP),
	}
	go storage.cleanup()
	return storage
}

// Get retrieves the request count for an IP as []byte (Fiber Storage interface)
func (s *IPBanStorage) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if IP is banned
	if ban, exists := s.bans[key]; exists {
		if time.Now().Before(ban.BannedUntil) {
			return []byte("999999"), nil
		}
	}

	// Count requests in the last second
	if timestamps, exists := s.requests[key]; exists {
		now := time.Now()
		count := 0
		for _, ts := range timestamps {
			if now.Sub(ts) <= 1*time.Second {
				count++
			}
		}
		return []byte(strconv.Itoa(count)), nil
	}

	return []byte("0"), nil
}

// Set increments the request count for an IP (Fiber Storage interface)
func (s *IPBanStorage) Set(key string, _ []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if IP is already banned
	if ban, exists := s.bans[key]; exists {
		if time.Now().Before(ban.BannedUntil) {
			return nil
		}
	}

	now := time.Now()
	if _, exists := s.requests[key]; !exists {
		s.requests[key] = make([]time.Time, 0)
	}

	s.requests[key] = append(s.requests[key], now)

	// Count requests in the last second
	count := 0
	for _, ts := range s.requests[key] {
		if now.Sub(ts) <= 1*time.Second {
			count++
		}
	}

	// Ban IP if exceeded limit (>10 requests per second)
	if count > 10 {
		s.bans[key] = &BannedIP{
			BannedUntil: now.Add(10 * time.Minute),
		}
	}

	return nil
}

// Delete removes an entry (required by Fiber Storage interface)
func (s *IPBanStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requests, key)
	delete(s.bans, key)
	return nil
}

// Reset clears all data (required by Fiber Storage interface)
func (s *IPBanStorage) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = make(map[string][]time.Time)
	s.bans = make(map[string]*BannedIP)
	return nil
}

// Close closes the storage (required by Fiber Storage interface)
func (s *IPBanStorage) Close() error {
	return nil
}

// cleanup removes expired data periodically
func (s *IPBanStorage) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()

		// Clean up old request timestamps
		for ip, timestamps := range s.requests {
			newTimestamps := make([]time.Time, 0)
			for _, ts := range timestamps {
				if now.Sub(ts) <= 1*time.Second {
					newTimestamps = append(newTimestamps, ts)
				}
			}
			if len(newTimestamps) == 0 {
				delete(s.requests, ip)
			} else {
				s.requests[ip] = newTimestamps
			}
		}

		// Clean up expired bans
		for ip, ban := range s.bans {
			if now.After(ban.BannedUntil) {
				delete(s.bans, ip)
			}
		}

		s.mu.Unlock()
	}
}

// IsBanned checks if an IP is currently banned
func (s *IPBanStorage) IsBanned(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if ban, exists := s.bans[ip]; exists {
		return time.Now().Before(ban.BannedUntil)
	}
	return false
}

var banStorage *IPBanStorage

// InitRateLimiter initializes the Fiber rate limiter with ban functionality
// Allows 10 requests per second, IP banned for 10 minutes on exceeding limit
func InitRateLimiter() fiber.Handler {
	if banStorage == nil {
		banStorage = NewIPBanStorage()
	}

	return limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			clientIP := c.IP()
			if banStorage.IsBanned(clientIP) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "ip banned",
					"message": "your IP has been temporarily banned for exceeding rate limits (10 minutes)",
				})
			}
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate limit exceeded",
				"message": "too many requests per second, please slow down",
			})
		},
		Storage: banStorage,
	})
}

// RateLimitMiddleware is the global rate limiter middleware
var RateLimitMiddleware = InitRateLimiter()
