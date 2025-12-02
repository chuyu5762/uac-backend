package service

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/pu-ac-cn/uac-backend/internal/model"
)

// Property 4: Client Secret 重置失效
// *For any* 应用，重置 Client Secret 后使用旧 Secret 进行认证应失败
// **Feature: unified-auth-center, Property 4: Client Secret 重置失效**
// **Validates: Requirements 2.5**
func TestProperty_ClientSecretResetInvalidatesOld(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	appNameGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "默认应用"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	properties.Property("重置后旧 Secret 失效新 Secret 有效", prop.ForAll(
		func(appName string) bool {
			appRepo := newMockAppRepository()
			orgRepo := newMockOrgRepository()
			svc := NewApplicationService(appRepo, orgRepo)
			orgSvc := NewOrganizationService(orgRepo)
			ctx := context.Background()

			org := &model.Organization{Name: "测试组织", Slug: "prop-test-org"}
			if err := orgSvc.Create(ctx, org); err != nil {
				return true
			}

			app := &model.Application{Name: appName, OrgID: org.ID}
			oldSecret, err := svc.Create(ctx, app)
			if err != nil {
				return true
			}

			newSecret, err := svc.ResetSecret(ctx, app.ID)
			if err != nil {
				t.Logf("重置失败: %v", err)
				return false
			}

			_, err = svc.ValidateClientCredentials(ctx, app.ClientID, oldSecret)
			if err == nil {
				t.Log("旧 Secret 应该失效")
				return false
			}

			_, err = svc.ValidateClientCredentials(ctx, app.ClientID, newSecret)
			if err != nil {
				t.Logf("新 Secret 验证失败: %v", err)
				return false
			}

			return true
		},
		appNameGen,
	))

	properties.TestingRun(t)
}
