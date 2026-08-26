package rocdate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToROC(t *testing.T) {
	tests := []struct {
		name      string
		input     time.Time
		wantVal   int
		wantStr   string
		wantErr   bool
		errTarget error
	}{
		{
			name:    "一般日期 2026-07-01",
			input:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			wantVal: 1150701,
			wantStr: "1150701",
		},
		{
			name:    "年初 2026-01-01",
			input:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			wantVal: 1150101,
			wantStr: "1150101",
		},
		{
			name:    "年末 2026-12-31",
			input:   time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			wantVal: 1151231,
			wantStr: "1151231",
		},
		{
			name:    "閏年 2028-02-29",
			input:   time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC),
			wantVal: 1170229,
			wantStr: "1170229",
		},
		{
			name:    "民國元年 1912-01-01",
			input:   time.Date(1912, 1, 1, 0, 0, 0, 0, time.UTC),
			wantVal: 10101,
			wantStr: "0010101",
		},
		{
			name:      "早於民國元年 1911-12-31",
			input:     time.Date(1911, 12, 31, 0, 0, 0, 0, time.UTC),
			wantErr:   true,
			errTarget: ErrBeforeROCYear,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToROC(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					assert.ErrorIs(t, err, tt.errTarget)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVal, got)
			assert.Equal(t, tt.wantStr, FormatROCString(got))
		})
	}
}

func TestFromROC(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    time.Time
		wantErr bool
	}{
		{
			name:  "一般日期 1150701",
			input: 1150701,
			want:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "閏年 1170229",
			input: 1170229,
			want:  time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "民國元年 0010101 (數值 10101)",
			input: 10101,
			want:  time.Date(1912, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "平年非法 2/29 1150229",
			input:   1150229,
			wantErr: true,
		},
		{
			name:    "非法月份 1151301",
			input:   1151301,
			wantErr: true,
		},
		{
			name:    "非法日數 1150431",
			input:   1150431,
			wantErr: true,
		},
		{
			name:    "數值太小 9999",
			input:   9999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromROC(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
