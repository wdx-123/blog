package system

import (
	"personal_blog/global"
	"personal_blog/internal/model/dto/request"
	serviceSystem "personal_blog/internal/service/system"
	"personal_blog/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ArticleCtrl 文章控制器
type ArticleCtrl struct {
	articleSvc *serviceSystem.ArticleSvc
}

// CreateArticle 创建文章
func (a *ArticleCtrl) CreateArticle(ctx *gin.Context) {
	var req request.ArticleCreateReq
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		global.Log.Error("绑定数据错误", zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusBadRequest).
			Failed("绑定数据错误", nil)
		return
	}
	err = a.articleSvc.ArticleCreate(ctx, &req)
	if err != nil {
		global.Log.Error("创建文章失败", zap.String("title", req.Title), zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusInternalServerError).
			Failed("创建文章失败", nil)
		return
	}
	// 发布
	response.NewResponse[any, any](ctx).
		SetCode(global.StatusOK).
		Success("文章已创建", map[string]any{
			"title":         req.Title,
			"category":      req.Category,
			"tags":          req.Tags,
			"visible_range": req.VisibleRange,
		})
}

// DeleteArticle 删除文章
func (a *ArticleCtrl) DeleteArticle(ctx *gin.Context) {
	// 1、获取请求结构体
	var req request.ArticleDeleteReq
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		global.Log.Error("绑定数据错误", zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusBadRequest).
			Failed("绑定数据错误", nil)
		return
	}
	// 2、删除文章
	err = a.articleSvc.ArticleDelete(ctx, &req)
	if err != nil {
		global.Log.Error("删除文章失败", zap.Strings("ids", req.IDs), zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusInternalServerError).
			Failed("删除文章失败", nil)
		return
	}
	// 3、返回数据
	response.NewResponse[any, any](ctx).
		SetCode(global.StatusOK).
		Success("删除成功", map[string]any{
			"count": len(req.IDs),
		})
}

// ArticleUpdate 更新文章
func (a *ArticleCtrl) ArticleUpdate(ctx *gin.Context) {
	// 1、获取请求结构体
	var req request.ArticleUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		global.Log.Error("绑定数据错误", zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusBadRequest).
			Failed("绑定数据错误", nil)
		return
	}
	// 2、更新文章
	if err := a.articleSvc.UpdateArticle(ctx.Request.Context(), req); err != nil {
		global.Log.Error("更新文章失败", zap.String("id", req.ID), zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusInternalServerError).
			Failed("更新文章失败", nil)
		return
	}
	// 3、返回数据
	response.NewResponse[any, any](ctx).
		SetCode(global.StatusOK).
		Success("更新成功", map[string]any{
			"id":    req.ID,
			"title": req.Title,
		})
}

// ArticleList 获取文章列表
func (a *ArticleCtrl) ArticleList(ctx *gin.Context) {
	// 1、获取请求结构体
	var req request.ArticleListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		global.Log.Error("绑定数据错误", zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusBadRequest).
			Failed("绑定数据错误", nil)
		return
	}
	// 2、获取数据
	respData, err := a.articleSvc.GetArticleList(ctx.Request.Context(), req)
	if err != nil {
		global.Log.Error("获取文章列表失败", zap.Error(err))
		response.NewResponse[any, any](ctx).
			SetCode(global.StatusInternalServerError).
			Failed("获取文章列表失败", nil)
		return
	}
	// 3、返回数据
	response.NewResponse[any, any](ctx).
		SetCode(global.StatusOK).
		Success("获取成功", respData)
}
