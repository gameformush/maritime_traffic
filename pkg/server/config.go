package server

type Config struct {
	Address string
	Port    int
}

func NewServer(cfg Config) *Server {
	return &Server{
		Config: cfg,
	}
}
