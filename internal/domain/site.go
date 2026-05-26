package domain

// Site 表示一个导航站点条目。
type Site struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Glow        string `json:"glow"`
	Sort        int    `json:"sort"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// Stats 汇总首页统计信息。
type Stats struct {
	SiteCount     int    `json:"siteCount"`
	CategoryCount int    `json:"categoryCount"`
	Coverage      string `json:"coverage"`
}

// CategoryStat 表示单个分类下的站点数量。
type CategoryStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
