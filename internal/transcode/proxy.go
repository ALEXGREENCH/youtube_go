package transcode

import (
	"io"
	"net/http"
	"net/http/httptest"
)

type streamProxy struct {
	server *httptest.Server
	client *http.Client
	src    string
}

func newStreamProxy(client *http.Client, src string) (*streamProxy, error) {
	p := &streamProxy{
		client: client,
		src:    src,
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), r.Method, p.src, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		req.Header.Set("User-Agent", defaultUserAgent)
		req.Header.Set("Referer", defaultReferer)
		if rng := r.Header.Get("Range"); rng != "" {
			req.Header.Set("Range", rng)
		}
		resp, err := p.client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = io.Copy(w, resp.Body)
	})
	p.server = httptest.NewServer(handler)
	return p, nil
}

func (p *streamProxy) URL() string {
	return p.server.URL
}

func (p *streamProxy) Close() {
	if p.server != nil {
		p.server.Close()
	}
}
