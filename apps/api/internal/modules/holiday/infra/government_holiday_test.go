package infra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGovernmentHolidayHTTPClientFixtureContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"date":"2027-01-01","name":"元旦","isDayOff":true},{"date":"2027/01/02","name":"補班","isDayOff":false}]}`))
	}))
	defer server.Close()
	items, err := (&GovernmentHolidayHTTPClient{Endpoint: server.URL}).Fetch(context.Background(), 2027)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || !items[0].IsDayOff || items[1].IsDayOff || !items[1].HolidayDate.Equal(time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestGovernmentHolidayHTTPClientParsesMultiYearCSV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte("date,year,name,isholiday,holidaycategory,description\n20251231,2025,跨年,是,節日,\n20260101,2026,開國紀念日,是,節日,\n20260102,2026,補班日,否,補班,\n"))
	}))
	defer server.Close()

	items, err := (&GovernmentHolidayHTTPClient{Endpoint: server.URL}).Fetch(context.Background(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 holidays for 2026, got %d: %+v", len(items), items)
	}
	if items[0].HolidayDate.Year() != 2026 || !items[0].IsDayOff || items[1].IsDayOff {
		t.Fatalf("unexpected parsed holidays: %+v", items)
	}
}
