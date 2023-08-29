package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gozelle/color"
	"github.com/gozelle/humanize"
	"github.com/gozelle/logger/v2"
	"github.com/gozelle/resty"
	"net/url"
	"strings"
	"time"
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

func NewAgent(client *resty.Client, host *url.URL) *Agent {
	return &Agent{client: client, host: host}
}

type Agent struct {
	host        *url.URL
	client      *resty.Client
	debug       bool
	accepter    func(resp *resty.Response) (err error)
	eventLogger *logger.ZapEventLogger
}

func (a *Agent) SetLogger(logger *logger.ZapEventLogger) {
	a.eventLogger = logger
}

func (a *Agent) logger() *logger.ZapEventLogger {
	if a.eventLogger == nil {
		a.eventLogger = logger.WithSkip(logger.Logger("resty-agent"), 3)
	}
	return a.eventLogger
}

func defaultAccepter(resp *resty.Response) error {
	if resp.IsSuccess() {
		return nil
	}
	return fmt.Errorf("request error: %s", resp.Status())
}

func (a *Agent) SetAccepter(accepter func(resp *resty.Response) (err error)) {
	a.accepter = accepter
}

func (a *Agent) fork() *Agent {
	return &Agent{
		client: a.client,
		host:   a.host,
	}
}

func (a *Agent) url(uri string) string {
	return a.host.JoinPath(uri).String()
}

func (a *Agent) Debug() *Agent {
	aa := a.fork()
	aa.debug = true
	return aa
}

func (a *Agent) debugPrintRequest(req *resty.Request, ) {
	if !a.debug {
		return
	}
	
	info := &strings.Builder{}
	u, _ := url.PathUnescape(req.URL)
	info.WriteString(fmt.Sprintf("\n[%s] %s %s:", color.CyanString("REQUEST"), color.BlueString(req.Method), u))
	space := fmt.Sprintf("|%s", strings.Repeat("-", 3))
	if req.Header != nil && len(req.Header) > 0 {
		d, _ := json.Marshal(req.Header)
		info.WriteString(fmt.Sprintf("\n%sheader: %s", space, string(d)))
	}
	if req.Body != nil {
		var body string
		switch v := req.Body.(type) {
		case string:
			body = v
		case []byte:
			body = string(v)
		default:
			d, _ := json.Marshal(req.Body)
			body = string(d)
		}
		info.WriteString(fmt.Sprintf("\n%sbody: %s", space, body))
	}
	a.logger().Debug(info.String())
}

func (a *Agent) debugPrintResponse(req *resty.Request, resp *resty.Response, err error, cost time.Duration) {
	if !a.debug {
		return
	}
	
	info := &strings.Builder{}
	u, _ := url.PathUnescape(req.URL)
	info.WriteString(fmt.Sprintf("\n[%s][%s] %s %s:", color.YellowString("RESPONSE"), color.YellowString(cost.String()), color.BlueString(req.Method), u))
	space := fmt.Sprintf("|%s", strings.Repeat("-", 3))
	
	if resp != nil {
		var status string
		if resp.IsSuccess() {
			status = color.GreenString(resp.Status())
		} else {
			status = color.RedString(resp.Status())
		}
		info.WriteString(fmt.Sprintf("\n%sstatus: %s", space, status))
		info.WriteString(fmt.Sprintf("\n%scontent size: %s", space, humanize.Bytes(uint64(resp.Size()))))
		if resp.Size() > 0 {
			info.WriteString(fmt.Sprintf("\n%sdata: %s", space, resp.String()))
		}
	}
	
	if err != nil {
		info.WriteString(fmt.Sprintf("\n%serror: %s", space, color.RedString(err.Error())))
	}
	
	a.logger().Debug(info.String())
}

func (a *Agent) Request(ctx context.Context, method, uri string, opts ...Option) (b Binder) {
	
	var (
		req  *resty.Request
		resp *resty.Response
		err  error
	)
	now := time.Now()
	defer func() {
		a.debugPrintResponse(req, resp, err, time.Since(now))
		b.err = err
	}()
	
	req = a.client.R()
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
	
	req.Method = method
	req.URL = a.url(uri)
	
	a.debugPrintRequest(req)
	
	resp, err = req.Send()
	if err != nil {
		return
	}
	
	var accept = defaultAccepter
	if a.accepter != nil {
		accept = a.accepter
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
