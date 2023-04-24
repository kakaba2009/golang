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
