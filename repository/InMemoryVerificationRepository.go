package repository

import (
	"errors"
	"sync"
	"time"
)

type otpItem struct {
	code      string
	expiresAt time.Time
}

type memVerificationRepo struct {
	data sync.Map // Thread-safe map
}

func NewInMemoryVerificationRepo() VerificationRepository {
	repo := &memVerificationRepo{}

	// Optional: Background Janitor to clean up map every 10 mins
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			repo.data.Range(func(key, value interface{}) bool {
				item := value.(otpItem)
				if time.Now().After(item.expiresAt) {
					repo.data.Delete(key)
				}
				return true
			})
		}
	}()

	return repo
}

func (r *memVerificationRepo) Save(key string, code string, duration time.Duration) error {
	r.data.Store(key, otpItem{
		code:      code,
		expiresAt: time.Now().Add(duration),
	})
	return nil
}

func (r *memVerificationRepo) Get(key string) (string, error) {
	val, ok := r.data.Load(key)
	if !ok {
		return "", errors.New("code not found")
	}

	item := val.(otpItem)

	// Check Expiry (Lazy Delete)
	if time.Now().After(item.expiresAt) {
		r.data.Delete(key) // Clean it up now
		return "", errors.New("code expired")
	}

	return item.code, nil
}

func (r *memVerificationRepo) Delete(key string) error {
	r.data.Delete(key)
	return nil
}
