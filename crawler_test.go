package main

import "testing"

func Test_fileName(t *testing.T) {
	type args struct {
		link   string
		start  string
		outDir string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"Simple 1", args{"https://example.com/foo", "https://example.com", "out"}, "out/foo", false},
		{"Simple 2", args{"https://example.com", "https://example.com", "out"}, "out/index.html", false},
		{"Simple 3", args{"https://news.ycombinator.com/news", "https://news.ycombinator.com/", "out"}, "out/news", false},
		{"Simple 4", args{"https://example.com/from?site=some/website&something", "https://example.com", "out"}, "out/from?site=some%2Fwebsite&something", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fileName(tt.args.link, tt.args.start, tt.args.outDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("fileName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fileName() got = %v, want %v", got, tt.want)
			}
		})
	}
}
