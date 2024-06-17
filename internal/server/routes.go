package server

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Get("/announce", s.trackerServer.AnnounceHandler)

    apiGroup := s.App.Group("/api")
	apiGroup.Post("/upload", s.explorerServer.UploadHandler)
	apiGroup.Get("/download", s.explorerServer.DownloadHandler)
	apiGroup.Get("/list", s.explorerServer.ListHandler)
}
