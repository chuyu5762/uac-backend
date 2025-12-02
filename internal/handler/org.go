// Package handler HTTP 处理器
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
	"github.com/pu-ac-cn/uac-backend/internal/service"
	"github.com/pu-ac-cn/uac-backend/pkg/response"
)

// OrgHandler 组织管理处理器
type OrgHandler struct {
	orgService service.OrganizationService
}

// NewOrgHandler 创建组织管理处理器
func NewOrgHandler(orgSvc service.OrganizationService) *OrgHandler {
	return &OrgHandler{orgService: orgSvc}
}

// ListOrgs 获取组织列表
// GET /api/v1/orgs
func (h *OrgHandler) ListOrgs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := &repository.OrgFilter{
		Name:   c.Query("name"),
		Status: c.Query("status"),
	}

	pagination := &repository.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	orgs, total, err := h.orgService.List(c.Request.Context(), filter, pagination)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	list := make([]gin.H, len(orgs))
	for i, org := range orgs {
		list[i] = h.orgToResponse(org)
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetOrg 获取组织详情
// GET /api/v1/orgs/:id
func (h *OrgHandler) GetOrg(c *gin.Context) {
	id := c.Param("id")
	org, err := h.orgService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeOrgNotFound, "组织不存在")
		return
	}

	response.Success(c, h.orgToResponse(org))
}

// CreateOrgRequest 创建组织请求
type CreateOrgRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Branding    *model.Branding `json:"branding"`
}

// CreateOrg 创建组织
// POST /api/v1/orgs
func (h *OrgHandler) CreateOrg(c *gin.Context) {
	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	org := &model.Organization{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.Branding != nil {
		org.Branding = *req.Branding
	}

	if err := h.orgService.Create(c.Request.Context(), org); err != nil {
		response.ErrorWithMsg(c, response.CodeServerError, err.Error())
		return
	}

	response.Success(c, h.orgToResponse(org))
}

// UpdateOrgRequest 更新组织请求
type UpdateOrgRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Branding    *model.Branding `json:"branding"`
	Status      string          `json:"status"`
}

// UpdateOrg 更新组织
// PUT /api/v1/orgs/:id
func (h *OrgHandler) UpdateOrg(c *gin.Context) {
	id := c.Param("id")
	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	org, err := h.orgService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeOrgNotFound, "组织不存在")
		return
	}

	if req.Name != "" {
		org.Name = req.Name
	}
	if req.Description != "" {
		org.Description = req.Description
	}
	if req.Branding != nil {
		org.Branding = *req.Branding
	}
	if req.Status != "" {
		org.Status = req.Status
	}

	if err := h.orgService.Update(c.Request.Context(), org); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, h.orgToResponse(org))
}

// DeleteOrg 删除组织
// DELETE /api/v1/orgs/:id
func (h *OrgHandler) DeleteOrg(c *gin.Context) {
	id := c.Param("id")

	if err := h.orgService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// UpdateBrandingRequest 更新品牌配置请求
type UpdateBrandingRequest struct {
	LogoURL      string `json:"logo_url"`
	FaviconURL   string `json:"favicon_url"`
	PrimaryColor string `json:"primary_color"`
	CustomCSS    string `json:"custom_css"`
}

// UpdateBranding 更新组织品牌配置
// PUT /api/v1/orgs/:id/branding
func (h *OrgHandler) UpdateBranding(c *gin.Context) {
	id := c.Param("id")
	var req UpdateBrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	branding := &model.Branding{
		LogoURL:      req.LogoURL,
		FaviconURL:   req.FaviconURL,
		PrimaryColor: req.PrimaryColor,
		CustomCSS:    req.CustomCSS,
	}

	if err := h.orgService.UpdateBranding(c.Request.Context(), id, branding); err != nil {
		response.ErrorWithMsg(c, response.CodeServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "品牌配置更新成功"})
}

// orgToResponse 将组织转换为响应格式
func (h *OrgHandler) orgToResponse(org *model.Organization) gin.H {
	return gin.H{
		"id":          org.ID,
		"tenant_id":   org.TenantID,
		"name":        org.Name,
		"slug":        org.Slug,
		"description": org.Description,
		"branding":    org.Branding,
		"status":      org.Status,
		"created_at":  org.CreatedAt,
		"updated_at":  org.UpdatedAt,
	}
}
