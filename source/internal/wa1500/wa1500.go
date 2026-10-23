package wa1500

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/html"
)

const (
	defaultBaseURL  = "https://wa1500.org"
	searchPath      = "/class.php"
	defaultTimeout  = 10 * time.Second
	defaultMaxBytes = 2 << 20
)

var ErrUpstream = errors.New("wa1500 upstream failure")

type Shooter struct {
	WAID      string `json:"waid"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Country   string `json:"country"`
	Club      string `json:"club"`
}

type Searcher interface {
	Search(ctx context.Context, query string) ([]Shooter, error)
}

type Client struct {
	baseURL  string
	http     *http.Client
	maxBytes int64
}

type ClientOption func(*Client)

func WithBaseURL(u string) ClientOption { return func(c *Client) { c.baseURL = u } }

func WithMaxBytes(n int64) ClientOption { return func(c *Client) { c.maxBytes = n } }

func WithTimeout(d time.Duration) ClientOption { return func(c *Client) { c.http.Timeout = d } }

func NewClient(opts ...ClientOption) *Client {
	c := &Client{baseURL: defaultBaseURL, http: &http.Client{Timeout: defaultTimeout}, maxBytes: defaultMaxBytes}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Search(ctx context.Context, query string) ([]Shooter, error) {
	form := "action=show&Suche=" + escapeLatin1(query) + "&Nation="
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+searchPath, strings.NewReader(form))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if int64(len(body)) > c.maxBytes {
		return nil, fmt.Errorf("%w: response too large", ErrUpstream)
	}
	shooters, err := Parse(strings.NewReader(latin1ToUTF8(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	return shooters, nil
}

func latin1ToUTF8(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		sb.WriteRune(rune(c))
	}
	return sb.String()
}

func escapeLatin1(s string) string {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xFF {
			r = '?'
		}
		b = append(b, byte(r))
	}
	return url.QueryEscape(string(b))
}

func Parse(r io.Reader) ([]Shooter, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var out []Shooter
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" && hasClass(n, "ranking") {
			if s, ok := parseRow(n); ok {
				out = append(out, s)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out, nil
}

func parseRow(tr *html.Node) (Shooter, bool) {
	cells := map[string]string{}
	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type == html.ElementNode && td.Data == "td" {
			if title := attr(td, "data-title"); title != "" {
				cells[title] = clean(text(td))
			}
		}
	}
	waid := cells["WA1500-ID"]
	if waid == "" || strings.IndexFunc(waid, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return Shooter{}, false
	}
	last, first, _ := strings.Cut(cells["Name"], ",")
	s := Shooter{WAID: waid, LastName: clean(last), FirstName: clean(first), Country: cells["Country"], Club: cells["Club"]}
	if s.LastName == "" {
		return Shooter{}, false
	}
	return s, true
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(text(c))
	}
	return sb.String()
}

func clean(s string) string {
	return strings.Join(strings.FieldsFunc(s, unicode.IsSpace), " ")
}
