package apperror

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestNewCoded_FormatsPer4_3(t *testing.T) {
	e := NewCoded("BAN", "DOWNLOAD", ClassS, "GRPC_TIMEOUT", "deadline exceeded", http.StatusInternalServerError)
	want := "BAN-DOWNLOAD-S-GRPC_TIMEOUT"
	if e.Code != want {
		t.Fatalf("Code = %q; want %q", e.Code, want)
	}
	if e.Class != "S" || e.Service != "BAN" || e.Feature != "DOWNLOAD" || e.Name != "GRPC_TIMEOUT" {
		t.Fatalf("structured fields not set: %+v", e)
	}
	if e.HTTPStatus != http.StatusInternalServerError {
		t.Fatalf("HTTPStatus = %d", e.HTTPStatus)
	}
}

func TestVBSFactories(t *testing.T) {
	v := V("URS", "REGISTER", "EMAIL_REQUIRED", "email is required")
	if v.Code != "URS-REGISTER-V-EMAIL_REQUIRED" || v.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("V() built unexpected: %+v", v)
	}

	b := B("URS", "USER", "USER_NOT_FOUND", "user not found", http.StatusNotFound)
	if b.Code != "URS-USER-B-USER_NOT_FOUND" || b.HTTPStatus != http.StatusNotFound {
		t.Fatalf("B() built unexpected: %+v", b)
	}

	cause := errors.New("connection refused")
	s := S("BAN", "DOWNLOAD", "GRPC_UNAVAILABLE", "downstream unavailable", cause)
	if s.Code != "BAN-DOWNLOAD-S-GRPC_UNAVAILABLE" || s.HTTPStatus != http.StatusInternalServerError {
		t.Fatalf("S() built unexpected: %+v", s)
	}
	if !errors.Is(s, cause) {
		t.Fatalf("S() didn't preserve cause: %v", s)
	}
}

func TestCodeRespects60CharLimit(t *testing.T) {
	// §4.3: SVC (3) + FEAT (≤10) + CLASS (1) + NAME (≤30) + 3 dashes = 47 max.
	e := NewCoded("URS", "REGISTER", ClassV, "EMAIL_FORMAT_INVALID_LONG_NAME", "msg", 400)
	if len(e.Code) > 60 {
		t.Fatalf("code exceeds 60 chars: %q (%d)", e.Code, len(e.Code))
	}
	if !strings.HasPrefix(e.Code, "URS-REGISTER-V-") {
		t.Fatalf("prefix wrong: %q", e.Code)
	}
}

func TestAs(t *testing.T) {
	original := V("URS", "LOGIN", "PASSWORD_REQUIRED", "password is required")
	wrapped := errors.New("layer 1: " + original.Error())
	_ = wrapped // wrapped isn't AppError-typed, so As should fail on it.

	if e, ok := As(original); !ok || e.Code != original.Code {
		t.Fatalf("As(original) = (%v, %v)", e, ok)
	}
	if _, ok := As(errors.New("plain error")); ok {
		t.Fatal("As(plainError) should be false")
	}
}
