package agent

import (
	"context"
	"github.com/gozelle/resty"
	"github.com/gozelle/spew"
	"github.com/gozelle/testify/require"
	"net/http"
	"net/url"
	"testing"
)

func TestAgent(t *testing.T) {
	
	host, err := url.Parse("http://api.m.taobao.com/")
	require.NoError(t, err)
	a := NewAgent(resty.New(), host)
	
	r := map[string]any{}
	err = a.Debug().Request(context.Background(), http.MethodPost, "/rest/api3.do?api=mtop.common.getTimestamp",
		WithRequestBody(map[string]any{
			"param": "OK",
		}),
		WithRequestHeader(map[string]string{
			"ContentType": "application/json",
		}),
		WithResponseFilter(func(resp *resty.Response) (data []byte, err error) {
			//fmt.Println(resp.String())
			return resp.Body(), nil
		}),
	).Bind(&r)
	require.NoError(t, err)
	spew.Json(r)
}
