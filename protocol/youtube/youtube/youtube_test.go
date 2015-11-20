package youtube

import (
	"net/http"
	"testing"
)

func TestYoutube(t *testing.T) {
	y, err := New("wZNYDzNGB-Q")
	if err != nil {
		t.Fatal(err)
	}
	url := y.Formats["43"]
	if url == "" {
		t.Fatal("expected format 43")
	}
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
