package server

import "testing"

func TestNormalizeExptime(t *testing.T) {
	now := int64(1_700_000_000)

	tests := []struct {
		name    string
		exptime int64
		want    int64
	}{
		{name: "zero is no expiration", exptime: 0, want: 0},
		{name: "relative seconds", exptime: 10, want: now + 10},
		{name: "relative threshold boundary", exptime: exptimeRealtimeThreshold, want: now + exptimeRealtimeThreshold},
		{name: "absolute unix time", exptime: exptimeRealtimeThreshold + 1, want: exptimeRealtimeThreshold + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeExptime(tt.exptime, now)
			if got != tt.want {
				t.Fatalf("normalizeExptime(%d, %d): want=%d got=%d", tt.exptime, now, tt.want, got)
			}
		})
	}
}
