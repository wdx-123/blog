package esMODEL

import (
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"personal_blog/internal/model/dto/request"
)

// Article 文章表
type Article struct {
	CreatedAt string `json:"created_at"` // 创建时间
	UpdatedAt string `json:"updated_at"` // 更新时间

	Cover        string   `json:"cover"`        // 文章封面
	Title        string   `json:"title"`        // 文章标题
	Keyword      string   `json:"keyword"`      // 文章标题-关键字
	Category     string   `json:"category"`     // 文章类别
	Tags         []string `json:"tags"`         // 文章标签
	Abstract     string   `json:"abstract"`     // 文章简介
	Content      string   `json:"content"`      // 文章内容
	VisibleRange uint     `json:"visibleRange"` // 可见范围 1-"全部可见"/2-"仅我可见"

	Views    int `json:"views"`    // 浏览量
	Comments int `json:"comments"` // 评论量
	Likes    int `json:"likes"`    // 收藏量
}

// EsOption 搜索参数
type EsOption struct {
    request.PageInfo
    Index          string          // 指定要查询的Elasticsearch索引
    Request        *search.Request // Elasticsearch的搜索请求，包含查询条件、排序、高亮等
    SourceIncludes []string        // 指定返回的字段，类似于SQL中的select指定列
    IncludeContent bool            // 是否返回 content 字段，默认不返回
}

// ArticleIndex 文章 ES 索引
func ArticleIndex() string {
	return "article_index"
}

// ArticleMapping 文章 Mapping 映射
func ArticleMapping() *types.TypeMapping {
	return &types.TypeMapping{
		Properties: map[string]types.Property{
			"created_at": types.DateProperty{
				NullValue: nil,
				Format: func(s string) *string {
					return &s
				}("yyyy-MM-dd HH:mm:ss")},
			"updated_at": types.DateProperty{
				NullValue: nil,
				Format: func(s string) *string {
					return &s
				}("yyyy-MM-dd HH:mm:ss")},

			"cover":         types.TextProperty{},
			"title":         types.TextProperty{},
			"keyword":       types.KeywordProperty{},
			"category":      types.KeywordProperty{},
			"tags":          types.KeywordProperty{},
			"abstract":      types.TextProperty{},
			"content":       types.TextProperty{},
			"visible_range": types.KeywordProperty{},

			"views":    types.IntegerNumberProperty{},
			"comments": types.IntegerNumberProperty{},
			"likes":    types.IntegerNumberProperty{},
		},
	}
}

/*
作用 ：定义文章索引中每个字段的数据类型和搜索规则

### 字段类型说明
1.
   DateProperty（日期类型）

   - created_at 、 updated_at ：创建和更新时间
   - 格式： yyyy-MM-dd HH:mm:ss
   - 支持时间范围查询和排序
2.
   TextProperty（全文搜索类型）

   - cover 、 title 、 abstract 、 content ：封面、标题、摘要、内容
   - 支持分词和全文搜索
   - 用户搜索时会匹配这些字段
3.
   KeywordProperty（精确匹配类型）

   - keyword 、 category ：关键字、分类
   - tags ：标签数组
   - 不分词，支持精确匹配和聚合统计
4.
   IntegerNumberProperty（整数类型）

   - views 、 comments 、 likes ：浏览量、评论数、点赞数
   - 支持数值范围查询和排序
*/
