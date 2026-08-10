package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/name212/govalue"

	"github.com/name212/docker-proxy/pkg/utils/pool"
	"github.com/name212/docker-proxy/pkg/utils/request"
)

type (
	closeFunc         func(*http.Response, *http.Request)
	urlPreparatorFunc func(*url.URL) *url.URL
)

type DockerHTTPClient struct {
	transport   http.RoundTripper
	clientsPool *pool.Pool[*http.Client]

	dockerServer  string
	urlPreparator urlPreparatorFunc

	logger *Logger
}

func NewDockerHTTPClient(dockerServer string, logger *Logger) (*DockerHTTPClient, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("http.DefaultTransport is not *http.Transport")
	}

	resultTransport := defaultTransport.Clone()
	defaultDialContext := defaultTransport.DialContext

	var urlPreparator urlPreparatorFunc

	host, port, err := net.SplitHostPort(dockerServer)
	if govalue.IsNil(err) {
		logger.Info(
			"Docker server parsed as host:port, request via tcp",
			logger.StringArg("host", host),
			logger.StringArg("port", port),
		)

		urlPreparator = func(u *url.URL) *url.URL {
			cpy := copyURL(u)
			cpy.Scheme = "http"
			cpy.Host = net.JoinHostPort(host, port)
			return cpy
		}

		resultTransport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return defaultDialContext(ctx, "tcp", dockerServer)
		}
	} else {
		logger.Info(
			"Cannot parse docker server as host:port, request via unix socket",
			logger.StringArg("unix", dockerServer),
		)

		urlPreparator = func(u *url.URL) *url.URL {
			cpy := copyURL(u)
			cpy.Scheme = "http"
			cpy.Host = "127.0.0.1"
			return cpy
		}

		resultTransport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return defaultDialContext(ctx, "unix", dockerServer)
		}
	}

	resultTransport.Proxy = http.ProxyFromEnvironment

	c := &DockerHTTPClient{
		transport:     resultTransport,
		logger:        logger,
		dockerServer:  dockerServer,
		urlPreparator: urlPreparator,
	}

	c.clientsPool = pool.NewPool(func() *http.Client {
		return &http.Client{
			Transport: c.transport,
		}
	})

	return c, nil
}

func (c *DockerHTTPClient) SendAndGetOnlyStatus(ctx context.Context, r *http.Request) error {
	response, cleanup, err := c.send(ctx, r)
	defer cleanup(response, r)

	if err != nil {
		return err
	}

	st := response.StatusCode

	if st >= 200 && st < 300 {
		return nil
	}

	return c.requestErr(r, "failed with incorrect status %d: '%s'", st, response.Status)
}

func (c *DockerHTTPClient) Send(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	response, cleanup, err := c.send(ctx, r)
	defer cleanup(response, r)

	if err != nil {
		return err
	}

	for name, values := range response.Header {
		w.Header().Set(name, values[0])
		for i := 1; i < len(values); i++ {
			w.Header().Add(name, values[i])
		}
	}

	w.WriteHeader(response.StatusCode)

	if govalue.IsNil(response.Body) {
		return nil
	}

	n, err := io.Copy(w, response.Body)
	if err != nil {
		return c.requestErr(r, "failed copy response (written %d bytes): '%s'", n, err.Error())
	}

	c.logger.Response(response, DebugCtx, "response written")

	return nil
}

func (c *DockerHTTPClient) send(ctx context.Context, r *http.Request) (*http.Response, closeFunc, error) {
	cl := c.clientsPool.Get()
	if govalue.IsNil(cl) {
		return nil, c.noCloseResponse, c.requestErr(r, "got nil http client from pool")
	}

	oldURL := r.URL
	oldRequestURI := r.RequestURI
	defer func() {
		c.clientsPool.Put(cl)

		r.URL = oldURL
		r.RequestURI = oldRequestURI
	}()

	r.RequestURI = ""
	r.URL = c.urlPreparator(oldURL)

	c.logger.Request(r, DebugCtx, "send request to docker")

	response, err := cl.Do(r.WithContext(ctx))

	if !govalue.IsNil(err) {
		urlErr, ok := err.(*url.Error)
		if !ok {
			return nil, c.noCloseResponse, c.requestErr(r, "unexpected error %s", err.Error())
		}

		if urlErr.Timeout() {
			return nil, c.noCloseResponse, c.requestErr(r, "timeout")
		}

		return nil, c.noCloseResponse, c.requestErr(r, "%s", err.Error())
	}

	c.logger.Response(response, DebugCtx, "request successful sent")

	return response, c.closeResponseBody, nil
}

func (c *DockerHTTPClient) requestErr(r *http.Request, f string, args ...any) error {
	err := fmt.Sprintf("request %s via %s error: ", request.RequestStr(r), c.dockerServer)
	err += fmt.Sprintf(f, args...)
	return fmt.Errorf("%s", err)
}

func (c *DockerHTTPClient) noCloseResponse(*http.Response, *http.Request) {}

func (c *DockerHTTPClient) closeResponseBody(response *http.Response, request *http.Request) {
	if govalue.IsNil(response.Body) {
		return
	}

	if err := response.Body.Close(); err != nil {
		c.logger.Error("response body is not closed", err, request)
	}
}

func copyURL(u *url.URL) *url.URL {
	n, err := url.Parse(u.String())
	if err != nil {
		n = &url.URL{
			Scheme:      u.Scheme,
			Host:        u.Host,
			User:        u.User,
			Path:        u.Path,
			RawPath:     u.RawPath,
			RawQuery:    u.RawQuery,
			Opaque:      u.Opaque,
			Fragment:    u.Fragment,
			RawFragment: u.RawFragment,
			OmitHost:    u.OmitHost,
			ForceQuery:  u.ForceQuery,
		}
	}

	return n
}
