package fiberutil

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

type ReverseProxy struct {
	fasthttp.HostClient
}

func NewReverseProxy(target *url.URL) *ReverseProxy {
	return &ReverseProxy{
		fasthttp.HostClient{
			Addr: target.Host,
		},
	}
}

func (r *ReverseProxy) Handle(c *fiber.Ctx) error {
	req := c.Request()
	res := c.Response()
	req.Header.Add("X-Forwarded-For", c.Context().RemoteIP().String())
	for _, h := range hopHeaders {
		hv := string(c.Get(h))
		if hv == "" {
			continue
		}
		if h == "Te" && hv == "trailers" {
			// Issue 21096: tell backend applications that
			// care about trailer support that we support
			// trailers. (We do, but we don't go out of
			// our way to advertise that unless the
			// incoming client request thought it was
			// worth mentioning)
			continue
		}
		req.Header.Del(h)
	}
	req.SetHost(r.Addr)
	err := r.Do(req, res)
	if err == nil {
		for _, h := range hopHeaders {
			res.Header.Del(h)
		}
	}
	return err
}
