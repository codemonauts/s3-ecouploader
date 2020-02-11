package main

import "testing"

type TestCase struct {
	Path string
	Src  string
	Dest string
	Key  string
}

func TestBuildRemotePath(t *testing.T) {
	testCases := []TestCase{
		{
			Path: "/mnt/data/test.jpg",
			Src:  "/mnt/data",
			Dest: "",
			Key:  "/test.jpg",
		}, {
			Path: "/mnt/data/foo/test.jpg",
			Src:  "/mnt/data",
			Dest: "/intern",
			Key:  "/intern/foo/test.jpg",
		}}

	for _, c := range testCases {
		remotePath := buildRemotePath(c.Path, c.Src, c.Dest)

		if remotePath != c.Key {
			t.Errorf("remotePath was incorrect, got: %s, want: %s.", remotePath, c.Key)
		}
	}
}
