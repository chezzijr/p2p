package api

type ExploreRequest struct {
	Search string `query:"search"`
	Offset int    `query:"offset"`
    Limit  int    `query:"limit"`
}
