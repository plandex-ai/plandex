package utils

import "testing"

func TestStripAddedBlankLines(t *testing.T) {
	tcs := []struct {
		name string
		orig string
		upd  string
		want string
	}{
		{
			name: "no change",
			orig: "a\nb\nc\n",
			upd:  "a\nb\nc\n",
			want: "a\nb\nc\n",
		},
		{
			name: "leading newline added",
			orig: "a\nb\n",
			upd:  "\n\na\nb\n",
			want: "a\nb\n",
		},
		{
			name: "trailing newline added",
			orig: "a\nb\n",
			upd:  "a\nb\n\n",
			want: "a\nb\n",
		},
		{
			name: "both ends, keep original padding",
			orig: "\nfoo\nbar\n\n",
			upd:  "\n\nfoo\nbar\n\n\n",
			want: "\nfoo\nbar\n\n",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := StripAddedBlankLines(tc.orig, tc.upd)
			if got != tc.want {
				t.Fatalf("\norig:\n%q\nupd:\n%q\nwant:\n%q\ngot:\n%q",
					tc.orig, tc.upd, tc.want, got)
			}
		})
	}
}
