package routes

import "time"

type request struct {
	url         string        `json:"url"`
	customShort string        `json:"short"`
	expiry      time.Duration `json:"expiry"`
}

type response struct {
	url             string        `json:"url"`
	customShort     string        `json:"short"`
	expiry          time.Duration `json:"expiry"`
	RateLimitRemain int           `json:"rate_limit"`
	RateLimitReset  time.Duration `json:"rate_limit_reset"`
}
