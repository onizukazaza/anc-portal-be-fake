package postgres

import "testing"

func TestMaskDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "standard DSN",
			dsn:  "postgres://user:secret@localhost:5432/mydb?sslmode=disable",
			want: "postgres://user:****@localhost:5432/mydb?sslmode=disable",
		},
		{
			name: "password with special chars (colon)",
			dsn:  "postgres://user:p%40ss%3Aword@localhost:5432/mydb?sslmode=disable",
			want: "postgres://user:****@localhost:5432/mydb?sslmode=disable",
		},
		{
			name: "no password",
			dsn:  "postgres://user@localhost:5432/mydb",
			want: "postgres://user@localhost:5432/mydb",
		},
		{
			name: "no user info",
			dsn:  "postgres://localhost:5432/mydb",
			want: "postgres://localhost:5432/mydb",
		},
		{
			name: "empty string",
			dsn:  "",
			want: "",
		},
		{
			name: "malformed URL",
			dsn:  "not-a-url",
			want: "not-a-url",
		},
		{
			name: "password with @ encoded",
			dsn:  "postgres://admin:p%40ssword@db.host:5432/prod?sslmode=require",
			want: "postgres://admin:****@db.host:5432/prod?sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskDSN(tt.dsn)
			if got != tt.want {
				t.Errorf("MaskDSN(%q)\n  got  = %q\n  want = %q", tt.dsn, got, tt.want)
			}
		})
	}
}
