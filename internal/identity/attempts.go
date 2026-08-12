package identity

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// attemptRecorder persiste intentos de login en login_attempts usando las
// funciones SECURITY DEFINER (la tabla no tiene SELECT/INSERT directos para
// consorcio_app porque los intentos se registran antes de conocer al usuario).
type attemptRecorder struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func (ar *attemptRecorder) countFailures(ctx context.Context, email, ip string, window time.Duration) (int64, error) {
	var n int64
	err := ar.pool.QueryRow(ctx,
		`SELECT app.count_recent_login_failures($1, $2, $3)`, email, ip, window,
	).Scan(&n)
	return n, err
}

func (ar *attemptRecorder) record(ctx context.Context, email, ip, source string, success bool) error {
	_, err := ar.pool.Exec(ctx,
		`SELECT app.record_login_attempt($1, $2, $3, $4)`, email, ip, source, success,
	)
	return err
}
