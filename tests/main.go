package tests

import (
	"fmt"
	"reflect"
)

type Student struct {
	Name string
}

func Test(i interface{}) {
	t := reflect.TypeOf(i)
	k := t.Kind()
	v := reflect.ValueOf(i)
	fmt.Printf("type: %s, kind: %s, value: %s", t, k, v)
}
