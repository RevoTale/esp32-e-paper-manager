package strictjson

import (
	"testing"
)

func TestValidate(t *testing.T) {
	for _, source := range []string{`{}`, `{"a":[1,true,null,{"b":"\\\"\/\ud83d\ude00"}]}`, `"Ї😀"`, `0`} {
		if err := Validate([]byte(source)); err != nil {
			t.Errorf("valid %s: %v", source, err)
		}
	}
	for _, source := range []string{`{"a":1,"a":2}`, `{"a":1,"\u0061":2}`, `{"x":[{"a":1,"a":2}]}`, `{} {}`, ``, `{"a":}`, `"\ud800"`, `"\udc00"`, `"\ud800\u0041"`, `"\ud800x"`, "\"\xff\""} {
		if err := Validate([]byte(source)); err == nil {
			t.Errorf("accepted %q", source)
		}
	}
}

func TestDepthAndSize(t *testing.T) {
	source := "0"
	for range 33 {
		source = "[" + source + "]"
	}
	if Validate([]byte(source)) == nil {
		t.Fatal("deep accepted")
	}
	if Validate(make([]byte, 32769)) == nil {
		t.Fatal("oversize accepted")
	}
}
