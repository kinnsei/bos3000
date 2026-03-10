package types

// Money represents a monetary amount in the smallest currency unit
// (e.g., fen for CNY, cents for USD). Using int64 avoids floating-point
// precision issues in billing calculations.
type Money int64

// PhoneNumber represents an E.164 formatted phone number string
// (e.g., "+8613800138000").
type PhoneNumber string

// CeilDiv returns the ceiling of a / b for positive integers.
// Useful for billing block rounding (e.g., rounding call duration
// up to the next 6-second block).
func CeilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}
