package kv

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRateLimit(t *testing.T) {
	tbl := LocalTbl{
		Capacity:   10,
		RefillRate: 2.0,
	}

	// Generate a cookie
	cookie, err := uuid.NewV7()
	if err != nil {
		t.Errorf("error on generating uuid: %v", err)
	}

	tbl.CreateCookie(context.TODO(), cookie.String())

	// Check that the cookie exists
	record := tbl.Records[cookie.String()]
	if record == nil {
		t.Errorf("could not find cookie %s in records", cookie.String())
	}

	var hits int
	var minHits int = 10

TEN_LOOP:
	for i := range 10 {
		allowed, remaining, _, err := tbl.Allow(context.TODO(), cookie.String())
		if err != nil {
			t.Errorf("encoutered error on checking rate limit for cookie %s: %v", cookie.String(), err)
		}
		minHits = int(math.Min(float64(minHits), float64(remaining)))
		if allowed {
			t.Logf("allowed request for iteration %d", i+1)
			t.Logf("%d remaining after iter %d", remaining, i+1)
			t.Logf("minimum is %d", minHits)
			hits = hits + 1
			time.Sleep(time.Duration(float64(i) * float64(time.Second) / 10))
		} else {
			break TEN_LOOP
		}
	}

	if hits != 10 {
		t.Errorf("expected %d hits, got %d", 10, hits)
	}

	if minHits != 6 {
		t.Errorf("expected rate to be 6 at the lowest, got %d", minHits)
	}
}
