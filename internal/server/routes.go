package server

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Get("/announce", s.trackerServer.AnnounceHandler)
    s.App.Post("/upload", s.explorerServer.UploadHandler)
    s.App.Get("/download", s.explorerServer.DownloadHandler)
    s.App.Get("/list", s.explorerServer.ListHandler)
}
