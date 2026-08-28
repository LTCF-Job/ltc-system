package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

func TestRegionService_CreateValidation(t *testing.T) {
	repo := repository.NewRegionRepository(nil)
	svc := NewRegionService(repo, nil)
	ctx := context.Background()

	t.Run("Missing name", func(t *testing.T) {
		_, err := svc.CreateRegion(ctx, CreateRegionRequest{
			Name: "",
		}, uuid.New(), "admin", "127.0.0.1", "test-ua")
		assert.ErrorIs(t, err, ErrRegionNameRequired)
	})

	t.Run("Valid request structure", func(t *testing.T) {
		req := CreateRegionRequest{
			Name:        "新竹縣",
			Description: "新竹縣營運區域",
			Status:      "active",
			SortOrder:   1,
		}
		assert.NotEmpty(t, req.Name)
		assert.Equal(t, "active", req.Status)
	})
}
