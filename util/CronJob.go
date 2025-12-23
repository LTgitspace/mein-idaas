package util

import (
	"log"
	"mein-idaas/repository"
	"time"
)

func StartDailyCleanup(repo repository.RefreshTokenRepository) {
	go func() {
		for {
			now := time.Now()

			// 1. Calculate target time: Today at 12:00 PM
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())

			// 2. If 12:00 PM has already passed today, schedule for tomorrow
			if nextRun.Before(now) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// 3. Calculate exact duration to wait
			duration := nextRun.Sub(now)
			log.Printf("Next refresh token record cleanup scheduled in %v (at %v)\n", duration, nextRun.Format(time.Kitchen))

			// 4. Sleep until that time
			time.Sleep(duration)

			// 5. Run the cleanup task
			log.Println("Deleting expired tokens...")
			if err := repo.DeleteExpired(); err != nil {
				log.Printf("Clean up failed succesfully: %v\n", err)
			} else {
				log.Println("Clean up completed.")
			}

			// 6. Loop restarts immediately.
			// Since we just finished (approx 12:00 PM), the next loop calculation
			// will see that "Today 12:00 PM" is just passed or is now,
			// so it will correctly add 24h for the next run.
			// (Adding a tiny buffer sleep here is good practice to ensure we don't double-trigger)
			time.Sleep(1 * time.Second)
		}
	}()
}
