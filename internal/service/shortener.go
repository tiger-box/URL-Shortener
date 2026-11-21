package service

import "context"

type Repository interface {
	Save(ctx context.Context, code string, url string) error
	Get(ctx context.Context, code string) (string, error)
	Delete(ctx context.Context, code string) error
}
