package transport

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWireDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantErr   bool
	}{
		{
			name:      "有效純日期字串",
			input:     `"2026-09-03"`,
			wantYear:  2026,
			wantMonth: time.September,
			wantDay:   3,
		},
		{
			name:      "有效 RFC3339 日期字串",
			input:     `"2026-09-03T08:30:00Z"`,
			wantYear:  2026,
			wantMonth: time.September,
			wantDay:   3,
		},
		{
			name:    "JSON null",
			input:   `null`,
			wantNil: true,
		},
		{
			name:    "空字串",
			input:   `""`,
			wantNil: true,
		},
		{
			name:    "空白字串",
			input:   `"   "`,
			wantNil: true,
		},
		{
			name:    "非法格式字串報錯",
			input:   `"invalid-date"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWireDate([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantYear, got.Year())
				assert.Equal(t, tt.wantMonth, got.Month())
				assert.Equal(t, tt.wantDay, got.Day())
			}
		})
	}
}

func TestWireDate_UnmarshalJSON_And_ToTimePtr(t *testing.T) {
	t.Run("空字串反序列化後 toTimePtr 回傳 nil", func(t *testing.T) {
		var d wireDate
		err := json.Unmarshal([]byte(`""`), &d)
		require.NoError(t, err)
		assert.Nil(t, d.toTimePtr())
	})

	t.Run("null 反序列化指標為 nil", func(t *testing.T) {
		var wrapper struct {
			Date *wireDate `json:"date"`
		}
		err := json.Unmarshal([]byte(`{"date": null}`), &wrapper)
		require.NoError(t, err)
		assert.Nil(t, wrapper.Date.toTimePtr())
	})

	t.Run("正常日期反序列化並取得時間指標", func(t *testing.T) {
		var d wireDate
		err := json.Unmarshal([]byte(`"2026-10-15"`), &d)
		require.NoError(t, err)
		ptr := d.toTimePtr()
		require.NotNil(t, ptr)
		assert.Equal(t, 2026, ptr.Year())
		assert.Equal(t, time.October, ptr.Month())
		assert.Equal(t, 15, ptr.Day())
	})
}

func TestCreateVehicleRequest_Unmarshal(t *testing.T) {
	t.Run("四個日期為空字串時成功反序列化且時間指標為 nil", func(t *testing.T) {
		siteID := uuid.New()
		body := `{
			"plateNo": "BZG-7915",
			"siteId": "` + siteID.String() + `",
			"compulsoryInsuranceExpiry": "",
			"passengerInsuranceExpiry": "",
			"thirdPartyInsuranceExpiry": "",
			"lastInspectionDate": "",
			"wheelchairAccessible": true,
			"status": "active"
		}`

		var req CreateVehicleRequest
		err := json.Unmarshal([]byte(body), &req)
		require.NoError(t, err)
		assert.Equal(t, "BZG-7915", req.PlateNo)
		assert.Equal(t, siteID, *req.SiteID)

		input := req.toInput()
		assert.Nil(t, input.CompulsoryInsuranceExpiry)
		assert.Nil(t, input.PassengerInsuranceExpiry)
		assert.Nil(t, input.ThirdPartyInsuranceExpiry)
		assert.Nil(t, input.LastInspectionDate)
	})

	t.Run("四個日期為 null 時成功反序列化且時間指標為 nil", func(t *testing.T) {
		siteID := uuid.New()
		body := `{
			"plateNo": "BZG-7915",
			"siteId": "` + siteID.String() + `",
			"displayName": null,
			"brand": null,
			"model": null,
			"manufactureYm": null,
			"compulsoryInsuranceExpiry": null,
			"passengerInsuranceExpiry": null,
			"thirdPartyInsuranceExpiry": null,
			"lastInspectionDate": null,
			"wheelchairAccessible": true,
			"status": "active"
		}`

		var req CreateVehicleRequest
		err := json.Unmarshal([]byte(body), &req)
		require.NoError(t, err)
		assert.Equal(t, "BZG-7915", req.PlateNo)
		assert.Equal(t, siteID, *req.SiteID)

		input := req.toInput()
		assert.Nil(t, input.CompulsoryInsuranceExpiry)
		assert.Nil(t, input.PassengerInsuranceExpiry)
		assert.Nil(t, input.ThirdPartyInsuranceExpiry)
		assert.Nil(t, input.LastInspectionDate)
	})
}
