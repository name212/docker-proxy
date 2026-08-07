package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/name212/govalue"

	"github.com/name212/docker-proxy/internal/utils/pool"
	"github.com/name212/docker-proxy/internal/utils/request"
)

type (
	closeFunc func()
)

type DockerHTTPClient struct {
	transport   http.RoundTripper
	clientsPool *pool.Pool[*http.Client]

	dockerServer string
	logger       *Logger
}

func NewDockerHTTPClient(dockerServer string, logger *Logger) (*DockerHTTPClient, error) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("http.DefaultTransport is not *http.Transport")
	}

	resultTransport := defaultTransport.Clone()
	defaultDialContext := defaultTransport.DialContext

	host, port, err := net.SplitHostPort(dockerServer)
	if govalue.IsNil(err) {
		logger.Info(
			"Docker server parsed as host:port, request via tcp",
			logger.StringArg("host", host),
			logger.StringArg("port", port),
		)

		resultTransport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return defaultDialContext(ctx, "tcp", dockerServer)
		}
	} else {
		logger.Info(
			"Cannot parse docker server as host:port, request via unix socket",
			logger.StringArg("unix", dockerServer),
		)

		resultTransport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return defaultDialContext(ctx, "unix", dockerServer)
		}
	}

	resultTransport.Proxy = http.ProxyFromEnvironment

	c := &DockerHTTPClient{
		transport:    resultTransport,
		logger:       logger,
		dockerServer: dockerServer,
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
	defer cleanup()

	if err != nil {
		return err
	}

	st := response.StatusCode

	if st >= 200 && st < 300 {
		return nil
	}

	return c.requestErr(ctx, r, "failed with incorrect status %d: '%s'", st, response.Status)
}

func (c *DockerHTTPClient) Send(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	response, cleanup, err := c.send(ctx, r)
	defer cleanup()

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
		return c.requestErr(ctx, r, "failed copy response (written %d bytes): '%s'", n, err.Error())
	}

	c.logger.Response(response, DebugCtx, "response written")

	return nil
}

func (c *DockerHTTPClient) send(ctx context.Context, r *http.Request) (*http.Response, closeFunc, error) {
	cl := c.clientsPool.Get()
	if govalue.IsNil(cl) {
		return nil, noClose, c.requestErr(ctx, r, "got nil http client for pool")
	}

	defer func() {
		c.clientsPool.Put(cl)
	}()

	c.logger.Request(r, DebugCtx, "send request to docker")

	response, err := cl.Do(r.WithContext(ctx))

	if !govalue.IsNil(err) {
		urlErr, ok := err.(*url.Error)
		if !ok {
			return nil, noClose, c.requestErr(ctx, r, "unexpected error %s", err.Error())
		}

		if urlErr.Timeout() {
			return nil, noClose, c.requestErr(ctx, r, "timeout")
		}

		return nil, noClose, c.requestErr(ctx, r, "%s", err.Error())
	}

	closeFn := func() {
		if govalue.IsNil(response.Body) {
			return
		}

		if err := response.Body.Close(); err != nil {
			c.logger.Error("response body is not closed", err, r)
		}
	}

	c.logger.Response(response, DebugCtx, "request successful sent")

	return response, closeFn, nil
}

func (c *DockerHTTPClient) requestErr(ctx context.Context, r *http.Request, f string, args ...any) error {
	id := getRequestID(ctx)
	err := fmt.Sprintf("request [%s] %s via %s error: ", id, request.RequestStr(r), c.dockerServer)
	err = err + fmt.Sprintf(f, args...)
	return fmt.Errorf("%s", err)
}

func noClose() {}
