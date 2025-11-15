package system

import (
	"personal_blog/internal/controller"

	"github.com/gin-gonic/gin"
)

type ArticleRouter struct {
}

func (ArticleRouter) InitArticleRouter(Router *gin.RouterGroup) {
	articleRouter := Router.Group("article")

	articleCtrl := controller.ApiGroupApp.SystemApiGroup.GetArticleCtrl()
	{
		articleRouter.POST("create", articleCtrl.CreateArticle)   // 创建文章
		articleRouter.DELETE("delete", articleCtrl.DeleteArticle) // 删除文章
		articleRouter.PUT("update", articleCtrl.ArticleUpdate)    // 更新文章
		articleRouter.GET("list", articleCtrl.ArticleList)        // 获取文章列表
	}
}
