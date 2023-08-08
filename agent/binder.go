package agent

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Binder struct {
	err  error
	data []byte
}

func (b Binder) Bind(ptr any) error {
	if b.err != nil {
		return b.err
	}
	return b.bind(ptr)
}

func (b Binder) bind(ptr any) (err error) {
	t := reflect.ValueOf(ptr)
	if t.Kind() != reflect.Ptr {
		err = fmt.Errorf("bind method expect a pointer, got: %s kind: %s", t.Type(), t.Kind())
		return
	} else if t.IsNil() {
		err = fmt.Errorf("the ptr is nil, please pass &ptr")
		return
	}
	return json.Unmarshal(b.data, ptr)
}

func (b Binder) Error() error {
	return b.err
}
