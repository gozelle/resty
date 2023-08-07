package agent

import (
	"context"
	"fmt"
	"github.com/gozelle/resty"
	"path"
)

type Option func(r *options)

type options struct {
	requestBody     any
	requestHeader   map[string]string
	requestInjector func(req *resty.Request) error
	responseFilter  func(resp *resty.Response) (data []byte, err error)
}

func WithRequestBody(body any) Option {
	return func(r *options) {
		r.requestBody = body
	}
}

func WithRequestHeader(h map[string]string) Option {
	return func(r *options) {
		r.requestHeader = h
	}
}

func WithResponseFilter(fn func(resp *resty.Response) (data []byte, err error)) Option {
	return func(r *options) {
		r.responseFilter = fn
	}
}

func WithRequestInjector(fn func(req *resty.Request) error) Option {
	return func(r *options) {
		r.requestInjector = fn
	}
}

func WithAfterRequest(fn func(req *resty.Request) error) Option {
	return func(r *options) {
		r.requestInjector = fn
	}
}

func NewAgent(client *resty.Client, host string) *Agent {
	return &Agent{client: client, host: host}
}

type Agent struct {
	host   string
	client *resty.Client
	debug  bool
	accept func(resp *resty.Response) (err error)
}

func defaultAccept(resp *resty.Response) error {
	if resp.IsSuccess() {
		return nil
	}
	return fmt.Errorf("request error: %s", resp.Status())
}

func (a *Agent) SetAccept(accept func(resp *resty.Response) (err error)) {
	a.accept = accept
}

func (a *Agent) fork() *Agent {
	return &Agent{
		client: a.client,
		host:   a.host,
	}
}

func (a *Agent) url(url string) string {
	return path.Join(a.host, url)
}

func (a *Agent) Debug() *Agent {
	aa := a.fork()
	aa.debug = true
	return aa
}

func (a *Agent) Request(ctx context.Context, method, uri string, opts ...Option) (b Binder) {
	
	var err error
	defer func() {
		b.err = err
	}()
	
	req := a.client.R()
	req.SetContext(ctx)
	
	_opts := &options{}
	for _, v := range opts {
		v(_opts)
	}
	
	req.Body = _opts.requestBody
	if _opts.requestHeader != nil {
		for k, v := range _opts.requestHeader {
			req.SetHeader(k, v)
		}
	}
	
	if _opts.requestInjector != nil {
		err = _opts.requestInjector(req)
		if err != nil {
			return
		}
	}
	
	resp, err := req.Execute(method, uri)
	if err != nil {
		return
	}
	
	var accept = defaultAccept
	if a.accept != nil {
		accept = a.accept
	}
	
	err = accept(resp)
	if err != nil {
		return
	}
	
	data := resp.Body()
	if _opts.responseFilter != nil {
		data, err = _opts.responseFilter(resp)
		if err != nil {
			return
		}
	}
	b.data = data
	
	return
}
