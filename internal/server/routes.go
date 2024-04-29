package server

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Get("/announce", s.trackerServer.AnnounceHandler)
    s.App.Post("/upload", s.explorerServer.UploadHandler)
}
