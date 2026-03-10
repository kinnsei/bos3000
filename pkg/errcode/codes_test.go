package errcode

import (
	"errors"
	"regexp"
	"testing"

	"encore.dev/beta/errs"
)

func TestNewError(t *testing.T) {
	err := NewError(errs.NotFound, PrefixNotFound, "no route for prefix 86")

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatal("expected *errs.Error")
	}
	if e.Code != errs.NotFound {
		t.Fatalf("code = %v, want NotFound", e.Code)
	}
	if e.Message != "no route for prefix 86" {
		t.Fatalf("message = %q, want %q", e.Message, "no route for prefix 86")
	}

	d, ok := e.Details.(ErrDetails)
	if !ok {
		t.Fatalf("details type = %T, want ErrDetails", e.Details)
	}
	if d.BizCode != PrefixNotFound {
		t.Fatalf("biz_code = %q, want %q", d.BizCode, PrefixNotFound)
	}
	if d.Suggestion == "" {
		t.Fatal("suggestion should not be empty")
	}
}

var screamingSnake = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`)

func TestAllConstantsFormat(t *testing.T) {
	for _, code := range AllCodes {
		if !screamingSnake.MatchString(code) {
			t.Errorf("code %q does not match SCREAMING_SNAKE_CASE", code)
		}
	}
}

func TestSuggestions(t *testing.T) {
	for _, code := range AllCodes {
		s := suggestFor(code)
		if s == "" {
			t.Errorf("code %q has no suggestion", code)
		}
	}
}
