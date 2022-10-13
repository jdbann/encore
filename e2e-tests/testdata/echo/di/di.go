package di

import (
	"archive/zip"
	"context"
	"database/sql"
	"sync"

	"encore.dev/cron"
)

type Response struct {
	Msg string
}

var _ = cron.NewJob("repeating-one", cron.JobConfig{
	Title:    "Call One every 2 hours",
	Every:    2 * cron.Hour,
	Endpoint: One,
})

//encore:service
type Service struct {
	Msg string

	// Include various types to make sure the parser doesn't complain.
	mu   sync.Mutex
	once *sync.Once
	db   *sql.DB
	fn   func() *zip.Writer
}

//encore:api public path=/di/one
func (s *Service) One(ctx context.Context) error {
	return nil
}

//encore:api public path=/di/two
func (s *Service) Two(ctx context.Context) (*Response, error) {
	return &Response{Msg: s.Msg}, nil
}

func initService() (*Service, error) {
	return &Service{Msg: "Hello World"}, nil
}