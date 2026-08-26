package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

func TestValidateRegionCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"Valid lowercase", "miaoli", true},
		{"Valid with hyphen", "new-taipei", true},
		{"Valid with underscore", "hsinchu_city", true},
		{"Valid alphanumeric", "region1", true},
		{"Too short", "a", false},
		{"Contains uppercase (unnormalized)", "Taipei", false},
		{"Contains spaces", "new taipei", false},
		{"Contains special chars", "region@1", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateRegionCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegionService_CreateValidation(t *testing.T) {
	repo := repository.NewRegionRepository(nil)
	svc := NewRegionService(repo, nil)
	ctx := context.Background()

	t.Run("Missing code", func(t *testing.T) {
		_, err := svc.CreateRegion(ctx, CreateRegionRequest{
			Code: "",
			Name: "測試區",
		}, uuid.New(), "admin", "127.0.0.1", "test-ua")
		assert.ErrorIs(t, err, ErrRegionCodeRequired)
	})

	t.Run("Invalid code format", func(t *testing.T) {
		_, err := svc.CreateRegion(ctx, CreateRegionRequest{
			Code: "Invalid Code!",
			Name: "測試區",
		}, uuid.New(), "admin", "127.0.0.1", "test-ua")
		assert.ErrorIs(t, err, ErrInvalidRegionCode)
	})

	t.Run("Missing name", func(t *testing.T) {
		_, err := svc.CreateRegion(ctx, CreateRegionRequest{
			Code: "test_region",
			Name: "",
		}, uuid.New(), "admin", "127.0.0.1", "test-ua")
		assert.ErrorIs(t, err, ErrRegionNameRequired)
	})
}
