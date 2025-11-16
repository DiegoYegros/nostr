package nip23

import "testing"

func TestNormalizeFrontMatterDate(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		expected string
		ok       bool
	}{
		{
			name:     "with timezone offset",
			value:    "2024-10-11 00:00:00 -0300",
			expected: "1728615600",
			ok:       true,
		},
		{
			name:     "blank",
			value:    " ",
			expected: "",
			ok:       false,
		},
	}

	for _, tc := range cases {
		got, ok := normalizeFrontMatterDate(tc.value)
		if ok != tc.ok {
			t.Fatalf("%s: expected ok=%v got %v", tc.name, tc.ok, ok)
		}
		if got != tc.expected {
			t.Fatalf("%s: expected %q got %q", tc.name, tc.expected, got)
		}
	}
}
