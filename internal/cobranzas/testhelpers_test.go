package cobranzas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan: got %d dests, want %d", len(dest), len(r.values))
	}
	for i := range dest {
		if err := assignValue(dest[i], r.values[i]); err != nil {
			return err
		}
	}
	return nil
}

type fakeRows struct {
	rows []fakeRow
	idx  int
	err  error
}

func (r *fakeRows) Close()                                   {}
func (r *fakeRows) Conn() *pgx.Conn                         { return nil }
func (r *fakeRows) Err() error                               { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag            { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.rows) {
		return pgx.ErrNoRows
	}
	return r.rows[r.idx-1].Scan(dest...)
}
func (r *fakeRows) Values() ([]any, error) {
	if r.idx == 0 || r.idx > len(r.rows) {
		return nil, pgx.ErrNoRows
	}
	return r.rows[r.idx-1].values, nil
}
func (r *fakeRows) RawValues() [][]byte { return nil }

func stubUUID(label string) pgtype.UUID {
	sum := sha256.Sum256([]byte(label))
	encoded := hex.EncodeToString(sum[:16])
	canonical := fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
	var out pgtype.UUID
	_ = out.Scan(canonical)
	return out
}

func stubDate(value string) pgtype.Date {
	var out pgtype.Date
	_ = out.Scan(value)
	return out
}

func stubTime() pgtype.Timestamptz {
	var out pgtype.Timestamptz
	_ = out.Scan(time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC))
	return out
}

func assignValue(dest any, value any) error {
	switch d := dest.(type) {
	case *pgtype.UUID:
		switch v := value.(type) {
		case pgtype.UUID:
			*d = v
			return nil
		case string:
			return d.Scan(v)
		}
	case *pgtype.Date:
		switch v := value.(type) {
		case pgtype.Date:
			*d = v
			return nil
		case string:
			return d.Scan(v)
		}
	case *pgtype.Timestamptz:
		switch v := value.(type) {
		case pgtype.Timestamptz:
			*d = v
			return nil
		case time.Time:
			return d.Scan(v)
		}
	case *pgtype.Text:
		switch v := value.(type) {
		case pgtype.Text:
			*d = v
			return nil
		case string:
			return d.Scan(v)
		}
	case *pgtype.Numeric:
		switch v := value.(type) {
		case pgtype.Numeric:
			*d = v
			return nil
		case string:
			return d.Scan(v)
		case int64:
			return d.Scan(v)
		}
	case *string:
		if value == nil {
			*d = ""
			return nil
		}
		*d = fmt.Sprint(value)
		return nil
	case *int64:
		switch v := value.(type) {
		case int64:
			*d = v
			return nil
		case int32:
			*d = int64(v)
			return nil
		case int:
			*d = int64(v)
			return nil
		}
	case *int32:
		switch v := value.(type) {
		case int32:
			*d = v
			return nil
		case int64:
			*d = int32(v)
			return nil
		case int:
			*d = int32(v)
			return nil
		}
	case *bool:
		if v, ok := value.(bool); ok {
			*d = v
			return nil
		}
	case *time.Time:
		switch v := value.(type) {
		case time.Time:
			*d = v
			return nil
		case pgtype.Timestamptz:
			if v.Valid {
				*d = v.Time
				return nil
			}
		}
	}
	if scanner, ok := dest.(interface{ Scan(any) error }); ok {
		return scanner.Scan(value)
	}
	rv := reflect.ValueOf(dest)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() && rv.Elem().CanSet() {
		switch rv.Elem().Kind() {
		case reflect.String:
			rv.Elem().SetString(fmt.Sprint(value))
			return nil
		case reflect.Int, reflect.Int32, reflect.Int64:
			if n, ok := value.(int64); ok {
				rv.Elem().SetInt(n)
				return nil
			}
		}
	}
	return fmt.Errorf("unsupported scan dest %T", dest)
}

type scanRecorder struct{ calls int }

func (s *scanRecorder) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) { s.calls++; return pgconn.CommandTag{}, nil }
func (s *scanRecorder) Query(context.Context, string, ...interface{}) (pgx.Rows, error)          { return &fakeRows{}, nil }
func (s *scanRecorder) QueryRow(context.Context, string, ...interface{}) pgx.Row                { return fakeRow{err: pgx.ErrNoRows} }
