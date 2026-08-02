package notion

import "testing"

func TestParseNotionDate(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{
			name: "unset property",
			raw:  "",
			want: "",
		},
		{
			name: "utc timestamp written by the upsert path",
			raw:  "2026-08-02T10:30:00.000+00:00",
			want: "2026-08-02 10:30",
		},
		{
			name: "zulu timezone",
			raw:  "2026-08-02T10:30:00.000Z",
			want: "2026-08-02 10:30",
		},
		{
			name: "non-utc workspace offset keeps its wall clock",
			raw:  "2026-08-02T10:30:00.000-04:00",
			want: "2026-08-02 10:30",
		},
		{
			name: "no fractional seconds",
			raw:  "2026-08-02T10:30:00+00:00",
			want: "2026-08-02 10:30",
		},
		{
			name: "microsecond precision",
			raw:  "2026-08-02T10:30:00.123456Z",
			want: "2026-08-02 10:30",
		},
		{
			name: "date without a time",
			raw:  "2026-08-02",
			want: "2026-08-02 00:00",
		},
		{
			name:    "unparseable value",
			raw:     "not a date",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseNotionDate(tc.raw)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseNotionDate(%q) = %q, want error", tc.raw, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseNotionDate(%q) unexpected error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Errorf("parseNotionDate(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
