package extract

import "testing"

func TestDefaultOutput(t *testing.T) {
	got := DefaultOutput("/tmp/movie.mp4")
	want := "/tmp/movie_clip.mp4"
	if got != want {
		t.Fatalf("DefaultOutput = %q, want %q", got, want)
	}
}
