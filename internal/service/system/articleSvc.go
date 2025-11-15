package system

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"personal_blog/global"
	"personal_blog/internal/model/consts"
	"personal_blog/internal/model/dto/request"
	resp "personal_blog/internal/model/dto/response"
	esModel "personal_blog/internal/model/elasticsearch"
	"personal_blog/internal/repository"
	"personal_blog/internal/repository/interfaces"
	"personal_blog/pkg/articleUtils"
	esUtil "personal_blog/pkg/elasticSearch"
	"personal_blog/pkg/imageUtils"
	"personal_blog/pkg/util"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

// ArticleSvc 文章服务
type ArticleSvc struct {
	articleRepo interfaces.ArticleRepository
}

// NewArticleSvc 创建文章服务实例
func NewArticleSvc(group *repository.Group) *ArticleSvc {
	return &ArticleSvc{
		articleRepo: group.SystemRepositorySupplier.GetArticleRepository(),
	}
}

func (a *ArticleSvc) ArticleCreate(
	ctx context.Context,
	req *request.ArticleCreateReq,
) error {
	// 1、通过关键字判断文章是存在
	b, err := articleUtils.Exists(ctx, req.Title)
	if err != nil {
		global.Log.Warn("检查标题失败，继续创建", zap.String("title", req.Title), zap.Error(err))
	}
	if b {
		global.Log.Warn("文章标题重复，继续创建", zap.String("title", req.Title))
	}
	// 2、设置结构体
	now := time.Now().Format("2006-01-02 15:04:05")
	articleToCreate := &esModel.Article{
		CreatedAt:    now,
		UpdatedAt:    now,
		Cover:        req.Cover,
		Title:        req.Title,
		Keyword:      req.Title,
		Category:     req.Category,
		Tags:         req.Tags,
		Abstract:     req.Abstract,
		Content:      req.Content,
		VisibleRange: req.VisibleRange,
	}
	// 3、在事物中创建文章，并更新相关消息
	return a.articleRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// 3.a 更新文章中的分类：新建文章仅需对新分类+1或创建
		if articleToCreate.Category != "" {
			if err = a.articleRepo.IncOrCreateCategory(ctx, tx, articleToCreate.Category); err != nil {
				global.Log.Error("更新分类计数失败",
					zap.String("category", articleToCreate.Category), zap.Error(err))
				return fmt.Errorf("更新分类计数失败: %v", err)
			}
		}
		// 3.b 更新标签计数
		if err = a.articleRepo.AddOrIncTag(ctx, tx, articleToCreate.Tags); err != nil {
			global.Log.Error("更新标签计数失败",
				zap.Strings("tags", articleToCreate.Tags), zap.Error(err))
			return fmt.Errorf("更新标签计数失败: %v", err)
		}
		// 3.c 创建文章
		err = articleUtils.CreateArticle(ctx, articleToCreate)
		if err != nil {
			global.Log.Error("创建文章失败",
				zap.String("title", articleToCreate.Title), zap.Error(err))
			return fmt.Errorf("创建文章失败: %v", err)
		}
		return nil
	})
}

// ArticleDelete 删除文章
func (a *ArticleSvc) ArticleDelete(
	ctx context.Context,
	req *request.ArticleDeleteReq,
) error {
	// 1、无数据，直接返回
	if len(req.IDs) == 0 {
		return nil
	}
	// 2、开启事物
	return a.articleRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// 3、逐个删除
		for _, id := range req.IDs {
			// 3.a 获取文章
			articleToDelete, err := esUtil.Get(ctx, id)
			if err != nil {
				global.Log.Warn("获取文章失败",
					zap.String("id", id), zap.Error(err))
				return fmt.Errorf("获取文章失败: %v", err)
			}
			// 3.b 删除文章类别
			if err = a.articleRepo.DecOrDeleteCategory(ctx, tx, articleToDelete.Category); err != nil {
				global.Log.Error("更新分类计数失败",
					zap.String("category", articleToDelete.Category), zap.Error(err))
				return fmt.Errorf("更新分类计数失败: %v", err)
			}
			// 3.c 删除标签
			if err = a.articleRepo.DecOrDeleteTag(ctx, tx, articleToDelete.Tags); err != nil {
				global.Log.Error("更新标签计数失败",
					zap.Strings("tags", articleToDelete.Tags), zap.Error(err))
				return fmt.Errorf("更新标签计数失败: %v", err)
			}
			// 3.d 修改所有图片
			// 3.d.1 初始化图片封面类别
			imageSlice := []string{articleToDelete.Cover}
			if err = imageUtils.InitImagesCategory(ctx, tx, imageSlice); err != nil {
				global.Log.Error("初始化图片封面类别失败",
					zap.Strings("urls", imageSlice), zap.Error(err))
				return fmt.Errorf("初始化图片封面类别失败: %v", err)
			}
			// 3.d.2 获取所有插图
			imageSlice, err = imageUtils.FindIllustrations(articleToDelete.Content)
			if err != nil {
				global.Log.Warn("解析插图失败",
					zap.String("id", id), zap.Error(err))
				return fmt.Errorf("解析插图失败: %v", err)
			}
			// 3.d.3 修改所有插图类别
			if err = imageUtils.ChangeImagesCategory(ctx, tx, imageSlice, consts.Category(0)); err != nil {
				global.Log.Error("修改插图类别失败",
					zap.Strings("urls", imageSlice), zap.Error(err))
				return fmt.Errorf("修改插图类别失败: %v", err)
			}
			// 3.d
			// 同时删除所有评论 todo
			// 3.e 删除文章
			var ids = []string{id}
			if err = esUtil.Delete(ctx, ids); err != nil {
				global.Log.Error("删除文章失败",
					zap.String("id", id), zap.Error(err))
				return fmt.Errorf("删除文章失败: %v", err)
			}
		}
		return nil
	})
}

// UpdateArticle 更新文章
func (a *ArticleSvc) UpdateArticle(
	ctx context.Context,
	req request.ArticleUpdateReq,
) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	articleToUpdate := struct {
		UpdatedAt    string   `json:"updated_at"`
		Cover        string   `json:"cover"`
		Title        string   `json:"title"`
		Keyword      string   `json:"keyword"`
		Category     string   `json:"category"`
		Tags         []string `json:"tags"`
		Abstract     string   `json:"abstract"`
		VisibleRange uint     `json:"visible_range"`
		Content      string   `json:"content"`
	}{
		UpdatedAt:    now,
		Cover:        req.Cover,
		Title:        req.Title,
		Keyword:      req.Title,
		Category:     req.Category,
		Tags:         req.Tags,
		Abstract:     req.Abstract,
		VisibleRange: req.VisibleRange,
		Content:      req.Content,
	}
	return a.articleRepo.Transaction(ctx, func(tx *gorm.DB) error {
		// 1、获取旧文章
		oldArticle, err := esUtil.Get(ctx, req.ID)
		if err != nil {
			global.Log.Warn("获取旧文章失败", zap.String("id", req.ID), zap.Error(err))
			return fmt.Errorf("获取旧文章失败: %v", err)
		}
		// 2、更新分类计数
		if err = a.updateCategoryCounts(ctx, tx, oldArticle.Category, articleToUpdate.Category); err != nil {
			return err
		}
		// 3、更新标签计数
		if err = a.updateTagCounts(ctx, tx, oldArticle.Tags, articleToUpdate.Tags); err != nil {
			return err
		}
		// 4、修改图片类别
		if err = updateCoverCategory(ctx, tx, oldArticle.Cover, articleToUpdate.Cover); err != nil {
			return err
		}
		// 5、修改插图类别
		if err = updateIllustrationsCategory(ctx, tx, oldArticle.Content, articleToUpdate.Content); err != nil {
			return err
		}
		// 6、更新文章
		if err = esUtil.Update(ctx, req.ID, articleToUpdate); err != nil {
			global.Log.Error("更新文章内容失败", zap.String("id", req.ID), zap.Error(err))
			return fmt.Errorf("更新文章内容失败: %v", err)
		}
		return nil
	})
}

// GetArticleList 文章列表
func (a *ArticleSvc) GetArticleList(
	ctx context.Context,
	info request.ArticleListReq,
) (res resp.ArticleListResp, err error) {
	// 1、ID查询
	if info.ID != nil && *info.ID != "" {
		// 1.a 按ID查询，查询到结构后，直接退出
		return a.articleListByID(ctx, *info.ID)
	}
	// 2、构建查询请求
	req := buildArticleSearchRequest(info)
	option := esModel.EsOption{
		PageInfo:       info.PageInfo,
		Index:          esModel.ArticleIndex(),
		Request:        req,
		IncludeContent: false,
	}
	// 3、分页查询
	hits, total, err := esUtil.EsPagination(ctx, option)
	if err != nil {
		return res, err
	}
	// 4、结果映射
	items, err := resp.FromHits(hits, false)
	if err != nil {
		global.Log.Warn("结果映射失败", zap.Error(err))
		return res, fmt.Errorf("结果映射失败: %v", err)
	}
	// 5、返回结果（包含分页元数据）
	// 5.a 第几页
	page := info.Page
	if page < 1 {
		page = 1
	}
	// 5.b 页大小
	pageSize := info.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	// 6、计算总页数
	totalPages := 0
	totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))

	res = resp.ArticleListResp{
		List:       items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
	return res, nil
}

// articleListByID 按ID查询文章
func (a *ArticleSvc) articleListByID(
	ctx context.Context,
	id string,
) (res resp.ArticleListResp, err error) {
	// 1、按ID查询
	esRes, gerr := global.ESClient.Get(esModel.ArticleIndex(), id).Do(ctx)
	if gerr != nil {
		global.Log.Warn("按ID查询失败",
			zap.String("id", id), zap.Error(gerr))
		return res, fmt.Errorf("按ID查询失败: %v", gerr)
	}
	// 2、判断结果
	if !esRes.Found { // 返回空值
		return resp.ArticleListResp{List: []resp.ArticleItemResp{}, Total: 0}, nil
	}
	// 3、结果映射
	var art esModel.Article
	if uerr := json.Unmarshal(esRes.Source_, &art); uerr != nil {
		global.Log.Warn("解析文章失败",
			zap.String("id", id), zap.Error(uerr))
		return res, fmt.Errorf("解析文章失败: %v", uerr)
	}
	// 4、结构转换
	item := resp.FromArticle(id, art, true)
	// 5、返回结果（按ID查询：单页单条）
	return resp.ArticleListResp{
		List: []resp.ArticleItemResp{item}, Total: 1, Page: 1, PageSize: 1, TotalPages: 1,
	}, nil
}

// buildArticleSearchRequest 构建搜索请求
func buildArticleSearchRequest(info request.ArticleListReq) *search.Request {
	// 1、创建搜索请求
	req := &search.Request{Query: &types.Query{}}
	// 2、设置查询条件(and查询)
	boolQuery := &types.BoolQuery{}
	// 3、设置标题
	if info.Title != nil && strings.TrimSpace(*info.Title) != "" {
		boolQuery.Must = append(
			boolQuery.Must,
			types.Query{Match: map[string]types.MatchQuery{"title": {Query: *info.Title}}})
	}
	// 4、设置摘要
	if info.Abstract != nil && strings.TrimSpace(*info.Abstract) != "" {
		boolQuery.Must = append(
			boolQuery.Must,
			types.Query{Match: map[string]types.MatchQuery{"abstract": {Query: *info.Abstract}}})
	}
	// 5、设置类别
	if info.Category != nil && strings.TrimSpace(*info.Category) != "" {
		boolQuery.Filter = []types.Query{{Term: map[string]types.TermQuery{"category": {Value: info.Category}}}}
	}
	// 6、设置标签
	if info.Tag != nil && strings.TrimSpace(*info.Tag) != "" {
		boolQuery.Filter = append(boolQuery.Filter, types.Query{Term: map[string]types.TermQuery{"tags": {Value: info.Tag}}})
	}
	// 7、设置可见范围
	if info.VisibleRange != 0 {
		boolQuery.Filter = append(boolQuery.Filter, types.Query{Term: map[string]types.TermQuery{"visible_range": {Value: info.VisibleRange}}})
	}
	// 8、根据需求，获取指定查询内容
	// 或者获取所有查询内容(带有时间排序)
	if boolQuery.Must != nil || boolQuery.Filter != nil {
		req.Query.Bool = boolQuery
	} else {
		req.Query.MatchAll = &types.MatchAllQuery{}
		req.Sort = []types.SortCombinations{types.SortOptions{SortOptions: map[string]types.FieldSort{"created_at": {Order: &sortorder.Desc}}}}
	}
	return req
}

/*

// must 会对搜索词分词处理 会进行评分，"golang"可能匹配到"go"、"golang教程"等
// filter 会对搜索词分词处理 不会进行评分，"golang"可能匹配到"golang"、"golang教程"等。 结果可缓存，查询更快
}
 "搜索场景分析": {
    "场景1_有搜索条件": {
      "描述": "当用户提供了标题、简介或类别等搜索条件时",
      "ES查询结构": {
        "query": {
          "bool": {
            "must": [
              {
                "match": {
                  "title": "用户输入的标题关键词"
                }
              },
              {
                "match": {
                  "abstract": "用户输入的简介关键词"
                }
              }
            ],
            "filter": [
              {
                "term": {
                  "category": "用户选择的类别"
                }
              }
            ]
          }
        }
      },
      "条件说明": [
        "must: 标题和简介必须匹配（计算相关性分数）",
        "filter: 类别必须精确匹配（不计算分数，性能更好）"
      ]
    }
*/

// updateCategoryCounts 分类计数调整：旧分类-1/删除，新分类+1/创建
func (a *ArticleSvc) updateCategoryCounts(
	ctx context.Context,
	tx *gorm.DB,
	oldCategory,
	newCategory string,
) error {
	if oldCategory == newCategory {
		return nil
	}
	if oldCategory != "" {
		if err := a.articleRepo.DecOrDeleteCategory(ctx, tx, oldCategory); err != nil {
			global.Log.Error("更新分类计数失败",
				zap.String("category", oldCategory), zap.Error(err))
			return fmt.Errorf("更新分类计数失败: %v", err)
		}
	}
	if newCategory != "" {
		if err := a.articleRepo.IncOrCreateCategory(ctx, tx, newCategory); err != nil {
			global.Log.Error("更新分类计数失败",
				zap.String("category", newCategory), zap.Error(err))
			return fmt.Errorf("更新分类计数失败: %v", err)
		}
	}
	return nil
}

// updateTagCounts 标签计数调整：新增+1/创建，删除-1/删除
func (a *ArticleSvc) updateTagCounts(
	ctx context.Context,
	tx *gorm.DB,
	oldTags,
	newTags []string,
) error {
	added, removed := util.DiffArrays(oldTags, newTags)
	if len(removed) > 0 {
		if err := a.articleRepo.DecOrDeleteTag(ctx, tx, removed); err != nil {
			global.Log.Error("更新标签计数失败",
				zap.Strings("tags", removed), zap.Error(err))
			return fmt.Errorf("更新标签计数失败: %v", err)
		}
	}
	if len(added) > 0 {
		if err := a.articleRepo.AddOrIncTag(ctx, tx, added); err != nil {
			global.Log.Error("更新标签计数失败",
				zap.Strings("tags", added), zap.Error(err))
			return fmt.Errorf("更新标签计数失败: %v", err)
		}
	}
	return nil
}

// updateCoverCategory 封面类别更新
func updateCoverCategory(
	ctx context.Context,
	tx *gorm.DB,
	oldCover,
	newCover string,
) error {
	if oldCover == newCover {
		return nil
	}
	if oldCover != "" {
		if err := imageUtils.InitImagesCategory(ctx, tx, []string{oldCover}); err != nil {
			global.Log.Error("初始化封面类别失败",
				zap.String("url", oldCover), zap.Error(err))
			return fmt.Errorf("初始化封面类别失败: %v", err)
		}
	}
	if newCover != "" {
		if err := imageUtils.ChangeImagesCategory(ctx, tx, []string{newCover}, consts.Category(4)); err != nil {
			global.Log.Error("更新封面类别失败",
				zap.String("url", newCover), zap.Error(err))
			return fmt.Errorf("更新封面类别失败: %v", err)
		}
	}
	return nil
}

// updateIllustrationsCategory 插图类别更新
func updateIllustrationsCategory(
	ctx context.Context,
	tx *gorm.DB,
	oldContent,
	newContent string,
) error {
	oldIllustrations, err := imageUtils.FindIllustrations(oldContent)
	if err != nil {
		global.Log.Warn("解析旧插图失败", zap.Error(err))
		oldIllustrations = nil
	}
	newIllustrations, err := imageUtils.FindIllustrations(newContent)
	if err != nil {
		global.Log.Warn("解析新插图失败", zap.Error(err))
		newIllustrations = nil
	}
	addedIllustrations, removedIllustrations := util.DiffArrays(oldIllustrations, newIllustrations)
	if len(removedIllustrations) > 0 {
		if err = imageUtils.InitImagesCategory(ctx, tx, removedIllustrations); err != nil {
			global.Log.Error("初始化插图类别失败",
				zap.Strings("urls", removedIllustrations), zap.Error(err))
			return fmt.Errorf("初始化插图类别失败: %v", err)
		}
	}
	if len(addedIllustrations) > 0 {
		if err = imageUtils.ChangeImagesCategory(ctx, tx, addedIllustrations, consts.Category(4)); err != nil {
			global.Log.Error("更新插图类别失败",
				zap.Strings("urls", addedIllustrations), zap.Error(err))
			return fmt.Errorf("更新插图类别失败: %v", err)
		}
	}
	return nil
}
