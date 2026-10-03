package dom

import "testing"

func TestDecodeEdits(t *testing.T) {
	for _, source := range []string{
		`{"edits":[{"id":"a","text":""}]}`,
		`{"edits":[{"id":"a","html":"<br>"},{"id":"b","attributes":{"title":null,"style":"color:red"}},{"id":"c","remove":true}]}`,
	} {
		if edits, err := DecodeEdits([]byte(source)); err != nil || len(edits) == 0 {
			t.Fatal(source, edits, err)
		}
	}
	for _, source := range []string{
		`null`, `[]`, `{}`, `{"edits":{}}`, `{"edits":[]}`, `{"edits":[null]}`,
		`{"edits":[],"other":0}`, `{"Edits":[]}`,
		`{"edits":[{"id":"a","id":"b","text":"x"}]}`,
		`{"edits":[{"id":"a","text":null}]}`,
		`{"edits":[{"id":"a","remove":false}]}`,
		`{"edits":[{"id":"a","attributes":null}]}`,
		`{"edits":[{"id":"a","attributes":{}}]}`,
		`{"edits":[{"id":"a","attributes":{"title":1}}]}`,
		`{"edits":[{"id":2,"text":"x"}]}`,
		`{"edits":[{"ID":"a","text":"x"}]}`,
		`{"edits":[{"id":"a","text":"\ud800"}]}`,
		`{"edits":[{"id":"a","text":true}]}`,
	} {
		if edits, err := DecodeEdits([]byte(source)); err == nil || edits != nil {
			t.Errorf("accepted %s", source)
		}
	}
}
