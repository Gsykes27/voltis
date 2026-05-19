package runtime

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type DevProxy struct {
	Target *url.URL
	RP     *httputil.ReverseProxy
}

func NewDevProxy(target string) (*DevProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	return &DevProxy{Target: u, RP: rp}, nil
}

func (p *DevProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.RP.ServeHTTP(w, r)
}

