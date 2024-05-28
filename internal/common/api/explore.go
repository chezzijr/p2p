package api

type ExploreRequestParam struct {
	Search string `query:"search"`
	Offset int    `query:"offset"`
    Limit  int    `query:"limit"`
}
