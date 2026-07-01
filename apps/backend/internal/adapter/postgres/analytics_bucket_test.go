package postgres

import "testing"

func TestBucketExpr(t *testing.T) {
	t.Run("accepts hour, day, week", func(t *testing.T) {
		for _, interval := range []string{"hour", "day", "week"} {
			expr, err := bucketExpr(interval)
			if err != nil {
				t.Fatalf("interval %q: unexpected error %v", interval, err)
			}
			want := "date_trunc('" + interval + "', request_started_at)"
			if expr != want {
				t.Fatalf("interval %q: expr = %q, want %q", interval, expr, want)
			}
		}
	})
	t.Run("rejects anything not on the allow-list", func(t *testing.T) {
		for _, bad := range []string{"", "month", "year", "day'; DROP TABLE llm_events; --"} {
			if _, err := bucketExpr(bad); err == nil {
				t.Fatalf("interval %q: want error, got nil", bad)
			}
		}
	})
}
