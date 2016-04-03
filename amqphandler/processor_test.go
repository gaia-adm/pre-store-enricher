package amqphandler

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnrichMessageEmptyByteSlice(t *testing.T) {
	in := make([]byte, 2)
	p := NewProcessor(1, "queue", "exchange")
	_, err := p.enrichMessage(&in)

	if err == nil {
		t.Error("expecting error when passing empty []byte: ", in)
	}
}

func TestEnrichMessageBadJson(t *testing.T) {
	in := []byte("bad json")
	p := NewProcessor(1, "queue", "exchange")
	_, err := p.enrichMessage(&in)

	if err == nil {
		t.Error("expecting err when passing a non json value: ", string(in))
	}
}

func TestEnrichMessageValidJson(t *testing.T) {
	in := []byte("{\"key\": \"value\"}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in)

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	var f interface{}
	err = json.Unmarshal(*out, &f)
	if err != nil {
		t.Fatal("failed to unmarshal the outout: ", string(*out), ", error is:", err)
	}

	m := f.(map[string]interface{})
	val, ok := m["gaia"]
	if !ok {
		t.Fatal("expecting the entry \"gaia\" to be presented after enrichment")
	}

	gaiamap := val.(map[string]interface{})
	val, ok = gaiamap["incoming_time"]
	if !ok {
		t.Fatal("expecting the key \"incoming_time\" to be presented under \"gaia\" json entry after enrichment")
	}

	_, err = time.Parse(time.RFC3339, val.(string))
	if err != nil {
		t.Fatal("incoming_time:", val.(string), " is not RFC3339 complient, err: ", err)
	}
}
