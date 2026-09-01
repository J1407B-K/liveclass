package memory

import (
	"context"

	"liveclass/internal/rpc/agent/dependency"
)

func postgresRead[T any](ctx context.Context, operation string, call func(context.Context) (T, error)) (T, error) {
	return dependency.Do(ctx, dependency.PostgresRead, operation, call)
}

func postgresWrite[T any](ctx context.Context, operation string, call func(context.Context) (T, error)) (T, error) {
	return dependency.Do(ctx, dependency.PostgresWrite, operation, call)
}

func postgresWriteError(ctx context.Context, operation string, call func(context.Context) error) error {
	_, err := postgresWrite(ctx, operation, func(callCtx context.Context) (struct{}, error) {
		return struct{}{}, call(callCtx)
	})
	return err
}
