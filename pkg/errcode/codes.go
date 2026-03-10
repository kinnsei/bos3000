package errcode

import "encore.dev/beta/errs"

// Business error code constants.
const (
	InsufficientBalance     = "INSUFFICIENT_BALANCE"
	BlacklistedNumber       = "BLACKLISTED_NUMBER"
	RateLimitExceeded       = "RATE_LIMIT_EXCEEDED"
	NoHealthyGateway        = "NO_HEALTHY_GATEWAY"
	InvalidCredentials      = "INVALID_CREDENTIALS"
	APIKeyRevoked           = "API_KEY_REVOKED"
	IPNotWhitelisted        = "IP_NOT_WHITELISTED"
	PrefixNotFound          = "PREFIX_NOT_FOUND"
	DailyLimitExceeded      = "DAILY_LIMIT_EXCEEDED"
	ConcurrencyLimitExceeded = "CONCURRENCY_LIMIT_EXCEEDED"
)

// AllCodes contains every defined business error code.
var AllCodes = []string{
	InsufficientBalance,
	BlacklistedNumber,
	RateLimitExceeded,
	NoHealthyGateway,
	InvalidCredentials,
	APIKeyRevoked,
	IPNotWhitelisted,
	PrefixNotFound,
	DailyLimitExceeded,
	ConcurrencyLimitExceeded,
}

// ErrDetails carries structured business error information.
// Implements the errs.ErrDetails marker interface.
type ErrDetails struct {
	BizCode    string `json:"biz_code"`
	Suggestion string `json:"suggestion"`
}

// ErrDetails implements the errs.ErrDetails marker interface.
func (ErrDetails) ErrDetails() {}

// NewError creates a structured *errs.Error with business error details.
func NewError(code errs.ErrCode, bizCode string, message string) error {
	return &errs.Error{
		Code:    code,
		Message: message,
		Details: ErrDetails{
			BizCode:    bizCode,
			Suggestion: suggestFor(bizCode),
		},
	}
}

func suggestFor(bizCode string) string {
	switch bizCode {
	case InsufficientBalance:
		return "Please top up your account balance"
	case BlacklistedNumber:
		return "This number is blocked; contact support if this is an error"
	case RateLimitExceeded:
		return "Please reduce request frequency and retry later"
	case NoHealthyGateway:
		return "All gateways are down; the system will retry automatically"
	case InvalidCredentials:
		return "Check your API key and secret"
	case APIKeyRevoked:
		return "Generate a new API key from the dashboard"
	case IPNotWhitelisted:
		return "Add your IP address to the allowlist in settings"
	case PrefixNotFound:
		return "No route configured for this number prefix"
	case DailyLimitExceeded:
		return "Daily sending limit reached; try again tomorrow or increase your limit"
	case ConcurrencyLimitExceeded:
		return "Too many concurrent calls; wait for some to complete"
	default:
		return ""
	}
}
