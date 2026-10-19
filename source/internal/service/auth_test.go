package service_test

import (
	"errors"
	"testing"

	"github.com/sollie/matchmark/internal/service"
)

func TestNewAdminAuthRefusesWeakPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"empty", "", true},
		{"fifteen chars", "123456789012345", true},
		{"sixteen chars", "1234567890123456", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.NewAdminAuth(nil, tc.password)
			if tc.wantErr && !errors.Is(err, service.ErrWeakPassword) {
				t.Fatalf("err = %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
