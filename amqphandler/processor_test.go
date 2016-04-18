package amqphandler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEmptyByteSlice(t *testing.T) {
	in := make([]byte, 2)
	p := NewProcessor(1, "queue", "exchange")
	_, err := p.enrichMessage(&in)

	if err == nil {
		t.Error("expecting error when passing empty []byte: ", in)
	}
}

func TestBadJson(t *testing.T) {
	in := []byte("bad json")
	p := NewProcessor(1, "queue", "exchange")
	_, err := p.enrichMessage(&in)

	if err == nil {
		t.Error("expecting err when passing a non json value: ", string(in))
	}
}

//Json.Unmarshal(v interface{}) converts numbers to float64 by default
//So when we marshal back the struct number's format can be changed
//(for instance: 1460888978000 can be converted into 1.460888978e+12)
//This bug was fixed by usage of encoding/json.Decoder.UseNumber()
//In this test we want to validate that the number format stays the same
func TestJsonNumber(t *testing.T) {
	jsonData := "\"time\":1460888978000"
	in := []byte("{" + jsonData + "}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in)

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	sOut := string(*out)
	if !strings.Contains(sOut, jsonData) {
		t.Fatal("original json was changed. Expected to see: ", jsonData, " inside output json, but output json was: ", sOut)
	}
}

//Validate that "gaia" json object was added to the original Json with valid attributes
func TestGaiaContent(t *testing.T) {
	in := []byte("{\"time\":1460888978000}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in)

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	//Unmarshal the output json and check "gaia" section exists with valid attributes
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
