package response

import (
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	esModel "personal_blog/internal/model/elasticsearch"
)

// ArticleItemResp 文章响应结构体（用于API返回）
// - 包含文章的基本信息、元数据和内容
// - 使用 `omitempty` 标签确保空值字段不会出现在JSON响应中
type ArticleItemResp struct {
	ID           string   `json:"id"`                // 文章唯一标识符
	CreatedAt    string   `json:"created_at"`        // 创建时间
	UpdatedAt    string   `json:"updated_at"`        // 最后更新时间
	Cover        string   `json:"cover"`             // 封面图片URL
	Title        string   `json:"title"`             // 文章标题
	Keyword      string   `json:"keyword"`           // 关键词/标签
	Category     string   `json:"category"`          // 文章分类
	Tags         []string `json:"tags"`              // 标签列表
	Abstract     string   `json:"abstract"`          // 文章摘要
	Content      string   `json:"content,omitempty"` // 文章正文内容（可选返回，空时省略）
	VisibleRange uint     `json:"visible_range"`     // 可见范围（0:私有 1:公开 2:部分可见）
	Views        int      `json:"views"`             // 浏览次数
	Comments     int      `json:"comments"`          // 评论数量
	Likes        int      `json:"likes"`             // 点赞数量
}

// ArticleListResp 文章列表响应
type ArticleListResp struct {
    List  []ArticleItemResp `json:"list"`
    Total int64             `json:"total"`
    Page  int               `json:"page"`
    PageSize int            `json:"page_size"`
    TotalPages int          `json:"total_pages"`
}

// FromArticle 将 ES 文档与 `_id` 映射为响应结构
// - includeContent=false 时不填充 Content 字段
func FromArticle(id string, a esModel.Article, includeContent bool) ArticleItemResp {
	item := ArticleItemResp{
		ID:           id,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		Cover:        a.Cover,
		Title:        a.Title,
		Keyword:      a.Keyword,
		Category:     a.Category,
		Tags:         a.Tags,
		Abstract:     a.Abstract,
		VisibleRange: a.VisibleRange,
		Views:        a.Views,
		Comments:     a.Comments,
		Likes:        a.Likes,
	}
	if includeContent {
		item.Content = a.Content
	}
	return item
}

// FromHit 将单个 ES Hit 映射为响应结构
// - includeContent 控制是否包含 Content 字段
func FromHit(hit types.Hit, includeContent bool) (ArticleItemResp, error) {
	// 1、将 Hit 的源数据转换为 Article
	var src esModel.Article
	if err := json.Unmarshal(hit.Source_, &src); err != nil {
		return ArticleItemResp{}, err
	}
	// 2、获取文档唯一标识_id
	id := ""
	if hit.Id_ != nil {
		id = *hit.Id_
	}
	// 3、返回响应结构
	return FromArticle(id, src, includeContent), nil
}

// FromHits 将多个 ES Hits 映射为响应结构切片
// - includeContent 控制是否包含 Content 字段
func FromHits(hits []types.Hit, includeContent bool) ([]ArticleItemResp, error) {
	// 1、创建一个空的响应切片
	out := make([]ArticleItemResp, 0, len(hits))
	// 2、遍历每个 Hit，将每个 Hit 转换为响应结构
	for _, h := range hits {
		// 3、转换数据
		it, err := FromHit(h, includeContent)
		if err != nil {
			return nil, err
		}
		// 4、放置转换后的响应结构
		out = append(out, it)
	}
	// 5、返回转换后的响应结构切片
	return out, nil
}
