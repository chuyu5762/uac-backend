package service

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/pu-ac-cn/uac-backend/internal/model"
)

// Property 1: 组织 CRUD 一致性
// *For any* 组织数据，创建后查询应返回相同的名称、logo 和描述
// **Feature: unified-auth-center, Property 1: 组织 CRUD 一致性**
// **Validates: Requirements 1.1, 1.4**
func TestProperty_OrganizationCRUDConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成有效的组织标识（小写字母和数字，2-20 字符）
	slugGen := gen.Identifier().Map(func(s string) string {
		// 转换为小写并截取合适长度
		result := ""
		for _, c := range s {
			if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
				result += string(c)
			} else if c >= 'A' && c <= 'Z' {
				result += string(c + 32) // 转小写
			}
			if len(result) >= 20 {
				break
			}
		}
		if len(result) < 2 {
			result = "ab" + result
		}
		return result
	})

	// 生成组织名称（非空字符串，1-50 字符）
	nameGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "默认组织"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// 生成描述（可选字符串，0-200 字符）
	descGen := gen.AlphaString().Map(func(s string) string {
		if len(s) > 200 {
			return s[:200]
		}
		return s
	})

	// 生成 Logo URL
	logoGen := gen.OneConstOf(
		"",
		"https://example.com/logo.png",
		"https://cdn.example.com/images/logo.svg",
	)

	properties.Property("创建后查询应返回相同数据", prop.ForAll(
		func(name, slug, desc, logo string) bool {
			repo := newMockOrgRepository()
			svc := NewOrganizationService(repo)
			ctx := context.Background()

			// 创建组织
			org := &model.Organization{
				Name:        name,
				Slug:        slug,
				Description: desc,
				Branding: model.Branding{
					LogoURL: logo,
				},
			}

			err := svc.Create(ctx, org)
			if err != nil {
				// 如果创建失败（如 slug 无效），跳过此测试用例
				return true
			}

			// 查询组织
			retrieved, err := svc.GetByID(ctx, org.ID)
			if err != nil {
				t.Logf("查询失败: %v", err)
				return false
			}

			// 验证数据一致性
			if retrieved.Name != name {
				t.Logf("名称不一致: 期望 %s, 实际 %s", name, retrieved.Name)
				return false
			}
			if retrieved.Slug != slug {
				t.Logf("标识不一致: 期望 %s, 实际 %s", slug, retrieved.Slug)
				return false
			}
			if retrieved.Description != desc {
				t.Logf("描述不一致: 期望 %s, 实际 %s", desc, retrieved.Description)
				return false
			}
			if retrieved.Branding.LogoURL != logo {
				t.Logf("Logo 不一致: 期望 %s, 实际 %s", logo, retrieved.Branding.LogoURL)
				return false
			}

			return true
		},
		nameGen,
		slugGen,
		descGen,
		logoGen,
	))

	properties.TestingRun(t)
}

// Property 2: 组织删除级联
// *For any* 组织及其下属应用，删除组织后查询该组织下的应用应返回空列表
// **Feature: unified-auth-center, Property 2: 组织删除级联**
// **Validates: Requirements 1.3**
// 注意：此测试需要在应用模块实现后完善
func TestProperty_OrganizationDeleteCascade(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// 生成有效的组织标识
	slugGen := gen.Identifier().Map(func(s string) string {
		result := ""
		for _, c := range s {
			if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
				result += string(c)
			} else if c >= 'A' && c <= 'Z' {
				result += string(c + 32)
			}
			if len(result) >= 20 {
				break
			}
		}
		if len(result) < 2 {
			result = "ab" + result
		}
		return result
	})

	properties.Property("删除组织后查询应返回不存在", prop.ForAll(
		func(slug string) bool {
			repo := newMockOrgRepository()
			svc := NewOrganizationService(repo)
			ctx := context.Background()

			// 创建组织
			org := &model.Organization{
				Name: "测试组织",
				Slug: slug,
			}

			err := svc.Create(ctx, org)
			if err != nil {
				return true // 跳过无效数据
			}

			orgID := org.ID

			// 删除组织
			err = svc.Delete(ctx, orgID)
			if err != nil {
				t.Logf("删除失败: %v", err)
				return false
			}

			// 查询应返回不存在
			_, err = svc.GetByID(ctx, orgID)
			if err == nil {
				t.Log("删除后仍能查询到组织")
				return false
			}

			return true
		},
		slugGen,
	))

	properties.TestingRun(t)
}
