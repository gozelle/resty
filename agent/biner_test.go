package agent

import (
	"github.com/gozelle/spew"
	"github.com/gozelle/testify/require"
	"testing"
)

func TestBindData(t *testing.T) {
	
	type User struct {
		Name string `json:"name"`
	}
	
	data := `{"name":"tom"}`
	
	var user *User
	
	b := Binder{
		err:  nil,
		data: []byte(data),
	}
	
	err := b.bind(&user)
	require.NoError(t, err)
	
	spew.Json(user)
}
