package articleUtils

import (
    "context"
    "github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
    "github.com/elastic/go-elasticsearch/v8/typedapi/types"
    "github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/refresh"
    "personal_blog/global"
    elasticsearch "personal_blog/internal/model/elasticsearch"
)

// CreateArticle 用于将文章创建到 Elasticsearch
func CreateArticle(ctx context.Context, a *elasticsearch.Article) error {
	// 将文章索引到Elasticsearch中，并设置刷新操作为 true
	_, err := global.ESClient.
		Index(elasticsearch.ArticleIndex()).
		Request(a).
		Refresh(refresh.True).
		Do(ctx)
	return err
}

// Exists 用于检查文章标题是否存在
func Exists(ctx context.Context, title string) (bool, error) {
	// 创建查询请求，匹配标题字段
	req := &search.Request{
		Query: &types.Query{
			Match: map[string]types.MatchQuery{"keyword": {Query: title}},
		},
	}
	// 执行搜索查询，查找是否存在该标题的文章
	res, err := global.ESClient.Search().
		Index(elasticsearch.ArticleIndex()). // 设置索引
		Request(req).                        // 设置查询请求
		Size(0).Do(ctx)                      // size为控制返回的命中数量,
	// （若size:0->1）不返回具体文档内容，只返回命中数量，性能更好
	if err != nil {
		return false, err
	}
	// 如果存在该标题，返回 true
	return res.Hits.Total.Value > 0, nil
}
