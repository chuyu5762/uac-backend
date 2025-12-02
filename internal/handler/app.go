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

// AppHandler 应用管理处理器
type AppHandler struct {
	appService service.ApplicationService
}

// NewAppHandler 创建应用管理处理器
func NewAppHandler(appSvc service.ApplicationService) *AppHandler {
	return &AppHandler{appService: appSvc}
}

// ListApps 获取应用列表
// GET /api/v1/apps
func (h *AppHandler) ListApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := &repository.AppFilter{
		OrgID: c.Query("org_id"),
		Name:  c.Query("name"),
	}

	pagination := &repository.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	apps, total, err := h.appService.List(c.Request.Context(), filter, pagination)
	if err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	list := make([]gin.H, len(apps))
	for i, app := range apps {
		list[i] = h.appToResponse(app)
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetApp 获取应用详情
// GET /api/v1/apps/:id
func (h *AppHandler) GetApp(c *gin.Context) {
	id := c.Param("id")
	app, err := h.appService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeAppNotFound, "应用不存在")
		return
	}

	response.Success(c, h.appToResponse(app))
}

// CreateAppRequest 创建应用请求
type CreateAppRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	OrgID         string   `json:"org_id"`
	RedirectURIs  []string `json:"redirect_uris"`
	AllowedScopes []string `json:"allowed_scopes"`
	OAuthMode     string   `json:"oauth_mode"`
}

// CreateApp 创建应用
// POST /api/v1/apps
func (h *AppHandler) CreateApp(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	var orgIDPtr *string
	if req.OrgID != "" {
		orgIDPtr = &req.OrgID
	}
	app := &model.Application{
		Name:          req.Name,
		Description:   req.Description,
		OrgID:         orgIDPtr, // 为空表示系统级应用
		RedirectURIs:  req.RedirectURIs,
		AllowedScopes: req.AllowedScopes,
		OAuthVersion:  req.OAuthMode,
	}

	if app.OAuthVersion == "" {
		app.OAuthVersion = model.OAuthVersion21
	}

	clientSecret, err := h.appService.Create(c.Request.Context(), app)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeServerError, err.Error())
		return
	}

	resp := h.appToResponse(app)
	resp["client_secret"] = clientSecret

	response.Success(c, resp)
}

// UpdateAppRequest 更新应用请求
type UpdateAppRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	RedirectURIs  []string `json:"redirect_uris"`
	AllowedScopes []string `json:"allowed_scopes"`
	OAuthMode     string   `json:"oauth_mode"`
	Status        string   `json:"status"`
}

// UpdateApp 更新应用
// PUT /api/v1/apps/:id
func (h *AppHandler) UpdateApp(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMsg(c, response.CodeInvalidRequest, "参数错误: "+err.Error())
		return
	}

	app, err := h.appService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeAppNotFound, "应用不存在")
		return
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.Description != "" {
		app.Description = req.Description
	}
	if req.RedirectURIs != nil {
		app.RedirectURIs = req.RedirectURIs
	}
	if req.AllowedScopes != nil {
		app.AllowedScopes = req.AllowedScopes
	}
	if req.OAuthMode != "" {
		app.OAuthVersion = req.OAuthMode
	}
	if req.Status != "" {
		app.Status = req.Status
	}

	if err := h.appService.Update(c.Request.Context(), app); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, h.appToResponse(app))
}

// DeleteApp 删除应用
// DELETE /api/v1/apps/:id
func (h *AppHandler) DeleteApp(c *gin.Context) {
	id := c.Param("id")

	if err := h.appService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, response.CodeServerError)
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// ResetSecret 重置 Client Secret
// POST /api/v1/apps/:id/reset-secret
func (h *AppHandler) ResetSecret(c *gin.Context) {
	id := c.Param("id")

	newSecret, err := h.appService.ResetSecret(c.Request.Context(), id)
	if err != nil {
		response.ErrorWithMsg(c, response.CodeServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"client_secret": newSecret})
}

// appToResponse 将应用转换为响应格式
func (h *AppHandler) appToResponse(app *model.Application) gin.H {
	var orgID any
	if app.OrgID != nil {
		orgID = *app.OrgID
	} else {
		orgID = nil
	}
	return gin.H{
		"id":             app.ID,
		"org_id":         orgID,
		"name":           app.Name,
		"description":    app.Description,
		"client_id":      app.ClientID,
		"redirect_uris":  app.RedirectURIs,
		"allowed_scopes": app.AllowedScopes,
		"oauth_mode":     app.OAuthVersion,
		"status":         app.Status,
		"created_at":     app.CreatedAt,
		"updated_at":     app.UpdatedAt,
	}
}
