package youtube

import (
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/mjibson/mog/codec"
	"github.com/mjibson/mog/codec/vorbis"
	"github.com/mjibson/mog/protocol"
	"github.com/mjibson/mog/protocol/youtube/youtube"
	"golang.org/x/oauth2"
)

func init() {
	protocol.Register("youtube", []string{"URL"}, New, reflect.TypeOf(&Youtube{}))
	gob.Register(new(Youtube))
}

func New(params []string, token *oauth2.Token) (protocol.Instance, error) {
	if len(params) != 1 {
		return nil, fmt.Errorf("expected one parameter")
	}
	y := Youtube{
		ID: params[0],
	}
	if _, err := y.Refresh(); err != nil {
		return nil, err
	}
	return &y, nil
}

type Youtube struct {
	ID  string
	URL string
}

func (y *Youtube) Key() string {
	return y.ID
}

func (y *Youtube) Refresh() (protocol.SongList, error) {
	yy, err := youtube.New(y.ID)
	if err != nil {
		return nil, err
	}
	streamURL := yy.Formats["43"]
	if streamURL == "" {
		return nil, fmt.Errorf("youtube: could not find format 43")
	}
	y.URL = streamURL
	return y.List()
}

func (y *Youtube) info() *codec.SongInfo {
	return &codec.SongInfo{
		Title: y.ID,
	}
}

func (y *Youtube) List() (protocol.SongList, error) {
	return protocol.SongList{
		codec.ID(y.ID): y.info(),
	}, nil
}

func (y *Youtube) Info(codec.ID) (*codec.SongInfo, error) {
	return y.info(), nil
}

func (y *Youtube) GetSong(codec.ID) (codec.Song, error) {
	return vorbis.NewSong(func() (io.ReadCloser, int64, error) {
		resp, err := http.Get(y.URL)
		if err != nil {
			return nil, 0, err
		}
		if resp.StatusCode != 200 {
			b, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			return nil, 0, fmt.Errorf("%v: %s", resp.Status, b)
		}
		return resp.Body, 0, nil
	})
}
