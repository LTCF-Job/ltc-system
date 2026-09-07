package main

import (
	"context"

	"github.com/google/uuid"
	identityapp "ltc-system/apps/api/internal/modules/identity/app"
)

type authUserStateChecker struct {
	admin identityapp.AdminIdentityProvider
}

func (c authUserStateChecker) Validate(ctx context.Context, actorID uuid.UUID, role string) (bool, error) {
	user, err := c.admin.GetUser(ctx, actorID)
	if err != nil {
		return false, err
	}
	if user == nil || user.Status != "active" {
		return false, nil
	}
	if user.RoleKey != "" {
		return user.RoleKey == role, nil
	}
	return user.Role == "" || user.Role == role, nil
}
