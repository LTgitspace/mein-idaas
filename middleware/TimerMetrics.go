package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TimerMetrics middleware tracks request duration and logs it
func TimerMetrics(c *fiber.Ctx) error {
	// Record start time
	startTime := time.Now()

	// Continue processing the request
	err := c.Next()

	// Calculate duration
	duration := time.Since(startTime)

	// Extract request details
	method := c.Method()
	path := c.Path()
	statusCode := c.Response().StatusCode()

	// Format duration in milliseconds for readability
	durationMs := duration.Milliseconds()

	// Log the metric
	log.Printf("[METRICS] %s %s - Status: %d - Duration: %dms (%.3fs)",
		method, path, statusCode, durationMs, duration.Seconds())

	// Return any error from processing
	return err
}

// TimerMetricsDetailed logs more detailed metrics including route name
func TimerMetricsDetailed(c *fiber.Ctx) error {
	// Record start time and memory stats
	startTime := time.Now()

	// Continue processing the request
	err := c.Next()

	// Calculate duration
	duration := time.Since(startTime)

	// Extract request details
	method := c.Method()
	path := c.Path()
	statusCode := c.Response().StatusCode()
	userID := c.Locals("user_id") // If user_id is stored in locals
	route := c.Route().Name       // Route name if set

	// Format duration
	durationMs := duration.Milliseconds()

	// Build log message
	logMsg := fmt.Sprintf("[METRICS] %s %s - Status: %d - Duration: %dms",
		method, path, statusCode, durationMs)

	if route != "" {
		logMsg += fmt.Sprintf(" - Route: %s", route)
	}

	if userID != nil {
		logMsg += fmt.Sprintf(" - User: %v", userID)
	}

	log.Printf("%s", logMsg)

	return err
}
