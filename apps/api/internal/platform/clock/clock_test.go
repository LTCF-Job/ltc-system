package clock

import (
	"testing"
	"time"
)

func TestSystemClockUsesAsiaTaipei(t *testing.T) {
	c := NewAsiaTaipei()
	now := c.Now()
	if now.Location().String() != "Asia/Taipei" {
		t.Fatalf("location = %q, want Asia/Taipei", now.Location())
	}
	if got := c.Today(); got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Fatalf("today = %v, want local midnight", got)
	}
	if got := gotDate(c.Today()); got != c.Now().Format("2006-01-02") {
		t.Fatalf("today date = %q, now date = %q", got, c.Now().Format("2006-01-02"))
	}
}

func gotDate(t time.Time) string { return t.Format("2006-01-02") }
