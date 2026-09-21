package dom

import (
	"encoding/json"

	"github.com/RevoTale/esp32-e-paper-manager/strictjson"
)

// DecodeEdits accepts exact field names. Null is permitted only for attribute
// deletion, never as an absent operation. Duplicate keys/invalid Unicode reject.
func DecodeEdits(source []byte) ([]Edit, error) {
	if err := strictjson.Validate(source); err != nil {
		return nil, editError()
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(source, &envelope) != nil || len(envelope) != 1 || envelope["edits"] == nil {
		return nil, editError()
	}
	var operations []map[string]json.RawMessage
	if json.Unmarshal(envelope["edits"], &operations) != nil || len(operations) == 0 || len(operations) > 64 {
		return nil, editError()
	}
	edits := make([]Edit, len(operations))
	for i, fields := range operations {
		if err := decodeEdit(fields, &edits[i]); err != nil {
			return nil, err
		}
	}
	return edits, nil
}

func decodeEdit(fields map[string]json.RawMessage, edit *Edit) error {
	for key, raw := range fields {
		if string(raw) == "null" {
			return editError()
		}
		if err := decodeField(edit, key, raw); err != nil {
			return editError()
		}
	}
	if edit.ID == "" || !validOperation(*edit) {
		return editError()
	}
	return nil
}

func decodeField(edit *Edit, key string, raw json.RawMessage) error {
	var err error
	switch key {
	case "id":
		err = json.Unmarshal(raw, &edit.ID)
	case "text":
		err = json.Unmarshal(raw, &edit.Text)
	case "html":
		err = json.Unmarshal(raw, &edit.HTML)
	case "attributes":
		err = json.Unmarshal(raw, &edit.Attributes)
	case "remove":
		err = json.Unmarshal(raw, &edit.Remove)
		if !edit.Remove {
			return editError()
		}
	default:
		return editError()
	}
	return err
}
