package parser

import "testing"

func TestFullTextSearch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single word >= 3 chars",
			input: "hello",
			want:  "+hello*",
		},
		{
			name:  "single word < 3 chars",
			input: "hi",
			want:  "hi*",
		},
		{
			name:  "multiple words all >= 3 chars",
			input: "foo bar baz",
			want:  "+foo +bar +baz*",
		},
		{
			name:  "short word excluded from prefix",
			input: "go is awesome",
			want:  "go is +awesome*",
		},
		{
			name:  "last word < 3 chars still gets suffix",
			input: "hello hi",
			want:  "+hello hi*",
		},
		{
			name:  "extra spaces between words",
			input: "foo   bar",
			want:  "+foo +bar*",
		},
		{
			name:  "lone operator token dropped",
			input: "SPX - Status 1",
			want:  "+SPX +Status 1*",
		},
		{
			name:  "operator chars stripped from words",
			input: "+foo -bar",
			want:  "+foo +bar*",
		},
		{
			name:  "embedded operator chars removed",
			input: "C++ a*b",
			want:  "C ab*",
		},
		{
			name:  "only operator chars yields empty",
			input: "+ - ~",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FullTextSearch(tt.input)
			if got != tt.want {
				t.Errorf("FullTextSearch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
