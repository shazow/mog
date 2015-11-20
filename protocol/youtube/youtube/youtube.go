package youtube

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/robertkrimen/otto"
)

var (
	ytplayerConfigRE = regexp.MustCompile(`;ytplayer\.config\s*=\s*({.*?});`)
	assetsRE         = regexp.MustCompile(`"assets":.+?"js":\s*("[^"]+")`)
	funcnameRE       = regexp.MustCompile(`\.sig\|\|([a-zA-Z0-9$]+)\(`)
	errNotYoutube    = fmt.Errorf("youtube: not a youtube video")
)

type Youtube struct {
	ID string
	// Formats is a map of format IDs to URLs.
	Formats map[string]string

	playerURL string
	cache     map[string][]byte
}

func (y *Youtube) URL() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "www.youtube.com",
		Path:   "/watch",
		RawQuery: url.Values{
			"v":            {y.ID},
			"gl":           {"US"},
			"hl":           {"en"},
			"has_verified": {"1"},
			"bpctr":        {"9999999999"},
		}.Encode(),
	}
}

func (y *Youtube) InfoURL() *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "www.youtube.com",
		Path:   "/get_video_info",
		RawQuery: url.Values{
			"video_id": {y.ID},
			"el":       {"info"},
			"ps":       {"default"},
			"gl":       {"US"},
			"hl":       {"en"},
		}.Encode(),
	}
}

func (y *Youtube) SignatureReplace(s string) (string, error) {
	purl := y.playerURL
	if strings.HasPrefix(purl, "//") {
		purl = "https:" + purl
	}
	b, err := y.get(purl)
	if err != nil {
		return "", err
	}
	bs := string(b)
	matches := funcnameRE.FindStringSubmatch(bs)
	if len(matches) == 0 {
		return "", fmt.Errorf("could not find funcname")
	}
	funcname := matches[1]
	fmt.Println("FUNCNAME", funcname)
	fpos := strings.Index(bs, "var " + funcname + "=function(")
	if fpos < 0 {
		return "", fmt.Errorf("func not found")
	}
	pos := strings.LastIndex(bs[:fpos], "};")
	total := 1
	i := pos - 1
	for ; i >= 0 && total > 0; i-- {
		switch bs[i] {
		case '}':
			total++
		case '{':
			total--
		}
	}
	if total != 0 {
		return "", fmt.Errorf("var not found")
	}
	vi := strings.LastIndex(bs[:i], "var ")
	if vi < 0 {
		return "", fmt.Errorf("var not found")
	}
	fi := strings.Index(bs[fpos:], "};")
	if fi < 0 {
		return "", fmt.Errorf("func not found")
	}
	fi += fpos + 2
	functext := bs[vi:fi]
	functext += fmt.Sprintf(`%s("%s");`, funcname, s)
	_, val, err := otto.Run(functext)
	if err != nil {
		return "", err
	}
	return val.ToString()
}

func get(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status: %v", resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}

func (y *Youtube) get(url string) ([]byte, error) {
	if _, ok := y.cache[url]; !ok {
		b, err := get(url)
		if err != nil {
			return nil, err
		}
		y.cache[url] = b
	}
	return y.cache[url], nil
}

func New(id string) (*Youtube, error) {
	y := &Youtube{
		ID:    id,
		cache: make(map[string][]byte),
	}
	defer func() {
		y.cache = nil
	}()
	b, err := get(y.URL().String())
	if err != nil {
		return nil, err
	}
	matches := ytplayerConfigRE.FindSubmatch(b)
	if matches == nil {
		return nil, errNotYoutube
	}
	var c youtubeConfig
	if err := json.Unmarshal(matches[1], &c); err != nil {
		return nil, err
	}
	matches = assetsRE.FindSubmatch(b)
	if matches == nil {
		return nil, errNotYoutube
	}
	if err := json.Unmarshal(matches[1], &y.playerURL); err != nil {
		return nil, err
	}
	info := c.Args
	if info.UrlEncodedFmtStreamMap == "" {
		return nil, fmt.Errorf("youtube: no stream_map present")
	}
	if info.Token == "" {
		return nil, fmt.Errorf("youtube: no token parameter")
	}
	encoded_url_map := info.UrlEncodedFmtStreamMap + "," + info.AdaptiveFmts
	y.Formats = make(map[string]string)
	for _, s := range strings.Split(encoded_url_map, ",") {
		v, err := url.ParseQuery(s)
		if err != nil {
			continue
		}
		if itag, url := v["itag"], v["url"]; len(itag) == 0 || len(url) == 0 {
			continue
		}
		format_id := v["itag"][0]
		u := v["url"][0]
		if s := v.Get("s"); s != "" {
			sig, err := y.SignatureReplace(s)
			if err != nil {
				return nil, err
			}
			u += "&signature=" + sig
		}
		y.Formats[format_id] = u
	}
	return y, nil
}

type videoInfo struct {
	UrlEncodedFmtStreamMap string `json:"url_encoded_fmt_stream_map"`
	Token                  string `json:"token"`
	AdaptiveFmts           string `json:"adaptive_fmts"`
}

type youtubeConfig struct {
	Args videoInfo `json:"args"`
}
