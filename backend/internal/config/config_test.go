package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "valid configuration",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/db",
				"API_PORT":     "8080",
				"APP_ENV":      "development",
				"APP_TIMEZONE": "America/Bogota",
			},
		},
		{
			name:    "Missing DATABASE_URL",
			env:     map[string]string{"DATABASE_URL": ""},
			wantErr: true,
		},
		{
			name: "Not numeric port",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/db",
				"API_PORT":     "not-a-number",
			},
			wantErr: true,
		},
		{
			name: "Invalid timezone",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/db",
				"APP_TIMEZONE": "Invalid/Timezone",
			},
			wantErr: true,
		},
		{
			name: "Missing ALLOWED_ORIGINS in production",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/db",
				"APP_ENV":      "production",
				"ALLOWED_ORIGINS": "",
			},
			wantErr: true,
		},
		{
			name: "Missing ALLOWED_ORIGINS in production",
			env: map[string]string{
				"DATABASE_URL": "postgres://localhost/db",
				"APP_ENV":      "production",
				"ALLOWED_ORIGINS": "http://app.biaenergy.com",
			},
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			for key, value := range tableTest.env {
				t.Setenv(key, value)
			}

			_, err := Load()

			if (err != nil) != tableTest.wantErr {
				t.Fatalf("Load() error %v, wantErr %v", err, tableTest.wantErr)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "An origin",
			input: "http://localhost:5173",
			want:  []string{"http://localhost:5173"},
		},
		{
			name: "With spaces",
			input: " http://a.com , http://b.com ",
			want: []string{"http://a.com", "http://b.com"},
		},
		{
			name: "Empty doesn't generate empty origin",
			input: "",
			want: []string{},
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			got := splitAndTrim(tableTest.input)

			if len(got) != len(tableTest.want) {
				t.Fatalf("len = %d, want %d (got %v)", len(got), len(tableTest.want), got)
			}

			for i := range got {
				if got[i] != tableTest.want[i] {
					t.Fatalf("got[%d] = %q, want %q", i, got[i], tableTest.want[i])
				}
			}
		})
	}
}
