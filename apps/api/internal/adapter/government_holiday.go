package adapter

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ltc-system/apps/api/internal/service"
)

const GovernmentHolidayCSVEndpoint = "https://data.ntpc.gov.tw/api/datasets/308dcd75-6434-45bc-a95f-584da4fed251/csv/file"

// GovernmentHolidayHTTPClient 下載並解析政府提供的行事曆資料。
// endpoint 可包含 {year}；未包含時保留原始 endpoint，因為資料集本身包含多個年份。
type GovernmentHolidayHTTPClient struct {
	Endpoint string
	Client   *http.Client
}

func (c *GovernmentHolidayHTTPClient) Fetch(ctx context.Context, year int) ([]service.HolidayRecord, error) {
	if c == nil || c.Endpoint == "" {
		return nil, fmt.Errorf("government holiday endpoint is empty")
	}
	endpoint := strings.ReplaceAll(c.Endpoint, "{year}", strconv.Itoa(year))
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create government holiday request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request government holiday API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("government holiday API returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read government holiday response: %w", err)
	}
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err == nil {
		return parseHolidayRecords(raw, year)
	}
	return parseHolidayCSV(body, year)
}

func parseHolidayCSV(body []byte, year int) ([]service.HolidayRecord, error) {
	records, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil || len(records) < 2 {
		return nil, fmt.Errorf("decode government holiday CSV: %w", err)
	}
	header := make(map[string]int, len(records[0]))
	for i, value := range records[0] {
		header[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))] = i
	}
	dateIndex, ok := csvColumn(header, "date", "日期")
	if !ok {
		return nil, fmt.Errorf("government holiday CSV is missing date column")
	}
	nameIndex, _ := csvColumn(header, "name", "節日")
	offIndex, _ := csvColumn(header, "isholiday", "isdayoff", "放假與否")
	result := make([]service.HolidayRecord, 0, len(records)-1)
	for _, record := range records[1:] {
		if dateIndex >= len(record) || strings.TrimSpace(record[dateIndex]) == "" {
			continue
		}
		dateValue := strings.TrimSpace(record[dateIndex])
		if len(dateValue) >= 4 && dateValue[:4] != strconv.Itoa(year) {
			continue
		}
		date, err := parseGovernmentDate(dateValue, year)
		if err != nil {
			return nil, err
		}
		name := ""
		if nameIndex >= 0 && nameIndex < len(record) {
			name = strings.TrimSpace(record[nameIndex])
		}
		isDayOff := true
		if offIndex >= 0 && offIndex < len(record) {
			isDayOff = parseBool(record[offIndex])
		}
		result = append(result, service.HolidayRecord{HolidayDate: date, Name: name, Source: "gov_calendar", IsDayOff: isDayOff})
	}
	return result, nil
}

func csvColumn(header map[string]int, names ...string) (int, bool) {
	for _, name := range names {
		if index, ok := header[strings.ToLower(name)]; ok {
			return index, true
		}
	}
	return -1, false
}

func parseHolidayRecords(raw interface{}, year int) ([]service.HolidayRecord, error) {
	if obj, ok := raw.(map[string]interface{}); ok {
		for _, key := range []string{"data", "records", "items", "result"} {
			if value, exists := obj[key]; exists {
				return parseHolidayRecords(value, year)
			}
		}
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("government holiday response must contain an array of records")
	}
	result := make([]service.HolidayRecord, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("government holiday record is not an object")
		}
		dateValue := firstString(obj, "date", "holidayDate", "holiday_date", "日期")
		date, err := parseGovernmentDate(dateValue, year)
		if err != nil {
			return nil, err
		}
		name := firstString(obj, "name", "description", "holidayName", "名稱")
		isDayOff := true
		if value, exists := firstValue(obj, "isDayOff", "isHoliday", "is_holiday", "休假"); exists {
			isDayOff = parseBool(value)
		}
		result = append(result, service.HolidayRecord{HolidayDate: date, Name: name, Source: "gov_calendar", IsDayOff: isDayOff})
	}
	return result, nil
}

func firstString(obj map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := obj[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}
func firstValue(obj map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if value, ok := obj[key]; ok {
			return value, true
		}
	}
	return nil, false
}
func parseBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return !strings.EqualFold(v, "false") && v != "0" && v != "否" && v != "上班"
	case float64:
		return v != 0
	}
	return true
}

func parseGovernmentDate(value string, year int) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "2006/01/02", "2006-1-2", "2006/1/2", "20060102"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			if parsed.Year() == year {
				return parsed, nil
			}
			return time.Time{}, fmt.Errorf("government holiday date %q is outside year %d", value, year)
		}
	}
	return time.Time{}, fmt.Errorf("unsupported government holiday date %q", value)
}
