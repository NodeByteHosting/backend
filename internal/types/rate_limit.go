package types

// RateLimitErrorResponse represents a rate limit exceeded error
type RateLimitErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error" example:"Rate limit exceeded. Maximum 5 requests per 15m0s. Retry after 900 seconds."`
	Code    string `json:"code" example:"RATE_LIMITED"`
}
