package service

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/pu-ac-cn/uac-backend/internal/model"
)

// Property 5: 用户-组织绑定访问控制
// *For any* 用户和组织，绑定后用户可访问该组织，解绑后访问应被拒绝
// **Feature: unified-auth-center, Property 5: 用户-组织绑定访问控制**
// **Validates: Requirements 3.3, 3.4, 3.5**
func TestProperty_UserOrgBindingAccessControl(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	usernameGen := gen.Identifier().Map(func(s string) string {
		if len(s) < 3 {
			return "usr" + s
		}
		if len(s) > 20 {
			return s[:20]
		}
		return s
	})

	properties.Property("绑定后有访问权限解绑后无权限", prop.ForAll(
		func(username string) bool {
			userRepo := newMockUserRepository()
			bindingRepo := newMockBindingRepository()
			orgRepo := newMockOrgRepository()
			svc := NewUserService(userRepo, bindingRepo, orgRepo)
			orgSvc := NewOrganizationService(orgRepo)
			ctx := context.Background()

			user := &model.User{Username: username, Email: username + "@test.com"}
			if err := svc.Create(ctx, user, "password123"); err != nil {
				return true
			}

			org := &model.Organization{Name: "测试组织", Slug: "prop-org-" + username}
			if err := orgSvc.Create(ctx, org); err != nil {
				return true
			}

			hasAccess, _ := svc.HasOrgAccess(ctx, user.ID, org.ID)
			if hasAccess {
				t.Log("绑定前不应该有访问权限")
				return false
			}

			if err := svc.BindOrganization(ctx, user.ID, org.ID); err != nil {
				t.Logf("绑定失败: %v", err)
				return false
			}

			hasAccess, _ = svc.HasOrgAccess(ctx, user.ID, org.ID)
			if !hasAccess {
				t.Log("绑定后应该有访问权限")
				return false
			}

			if err := svc.UnbindOrganization(ctx, user.ID, org.ID); err != nil {
				t.Logf("解绑失败: %v", err)
				return false
			}

			hasAccess, _ = svc.HasOrgAccess(ctx, user.ID, org.ID)
			if hasAccess {
				t.Log("解绑后不应该有访问权限")
				return false
			}

			return true
		},
		usernameGen,
	))

	properties.TestingRun(t)
}
