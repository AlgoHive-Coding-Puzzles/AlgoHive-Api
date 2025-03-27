package config

import "time"

// Rate limit configuration
type RateLimitConfig struct {
	AttemptsThreshold1 int           // Number of attempts before first cooldown
	CooldownDuration1  time.Duration // First cooldown duration
	AttemptsThreshold2 int           // Number of attempts before second cooldown
	CooldownDuration2  time.Duration // Second cooldown duration
}

var DefaultRateLimitConfig = RateLimitConfig{
	AttemptsThreshold1: 3,
	CooldownDuration1:  3 * time.Minute,
	AttemptsThreshold2: 5,
	CooldownDuration2:  5 * time.Minute,
}
