package global

type ConfigFile struct {
	Url      string `json:"url"`
	Threads  int    `json:"threads"`
	Interval int    `json:"interval"`
}

type Article struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

type ArticleData struct {
	Title       string
	ArticleList []string
}

type Record struct {
	Id    string
	Title string
	Url   string
}
