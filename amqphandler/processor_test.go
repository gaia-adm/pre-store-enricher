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
	_, err := p.enrichMessage(&in, "")

	if err == nil {
		t.Error("expecting error when passing empty []byte: ", in)
	}
}

func TestBadJson(t *testing.T) {
	in := []byte("bad json")
	p := NewProcessor(1, "queue", "exchange")
	_, err := p.enrichMessage(&in, "")

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
	out, err := p.enrichMessage(&in, "")

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
	in := []byte("{\"key\":\"val\"}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "") //Empty second param means we do not send instructions to extract event time from json

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	gaiaTime, ok := gaiamap["gaia_time"]
	if !ok {
		t.Fatal("expecting the key \"gaia_time\" to be presented under \"gaia\" json entry after enrichment")
	}

	eventTime, ok := gaiamap["event_time"]
	if !ok {
		t.Fatal("expecting the key \"event_time\" to be presented under \"gaia\" json entry after enrichment")
	}

	_, err = time.Parse(time.RFC3339, gaiaTime.(string))
	if err != nil {
		t.Fatal("gaia_time:", gaiaTime.(string), " is not RFC3339 complient, err: ", err)
	}

	_, err = time.Parse(time.RFC3339, eventTime.(string))
	if err != nil {
		t.Fatal("event_time:", eventTime.(string), " is not RFC3339 complient, err: ", err)
	}

	//Default value of eventTime in case it was not extracted from the json should be exactly the same as gaia_time
	if gaiaTime != eventTime {
		t.Fatal("gaia_time: ", gaiaTime.(string), " and event_time: ", eventTime.(string), " should be the same in case that event time was not extracted from json, but they are different")
	}
}

func TestEventTimeNoFieldLocation(t *testing.T) {
	in := []byte("{\"key\":\"val\"}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "") //Empty second param means we do not send instructions to extract event time from json

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	//We validate that event_time presented and has time value (which is now())
	eventTime, _ := gaiamap["event_time"]
	_, err = time.Parse(time.RFC3339, eventTime.(string))
	if err != nil {
		t.Fatal("event_time:", eventTime.(string), " is not RFC3339 complient, err: ", err)
	}
}

func TestEventTimeInvalidFieldLocation(t *testing.T) {
	in := []byte("{\"folder\": {\"time\":1460888978000}}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "folder.nonexistfield")

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	//We validate that event_time presented and has time value (which is now())
	eventTime, _ := gaiamap["event_time"]
	_, err = time.Parse(time.RFC3339, eventTime.(string))
	if err != nil {
		t.Fatal("event_time:", eventTime.(string), " is not RFC3339 complient, err: ", err)
	}
}

func TestEventTimeExtractingStringFormat(t *testing.T) {
	in := []byte("{\"folder\": {\"time\":\"2016-05-17T15:27:16+03:00\"}}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "folder.time")

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	//We validate that event_time presented and has time value that equals to the input
	eventTime, _ := gaiamap["event_time"]
	if eventTime.(string) != "2016-05-17T15:27:16+03:00" {
		t.Fatal("event contained the time 2016-05-17T15:27:16+03:00, but event_time was: ", eventTime.(string))
	}
}

func TestEventTimeExtractingUnixTimeSecFormat(t *testing.T) {
	in := []byte("{\"folder\": {\"time\":1460888978}}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "folder.time")

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	//We validate that event_time presented and has time value that equals to the input
	//(in RFC3339 format)
	eventTime, _ := gaiamap["event_time"]
	if eventTime.(string) != "2016-04-17T10:29:38Z" {
		t.Fatal("event contained the time 2016-04-17T10:29:38Z, but event_time was: ", eventTime.(string))
	}
}

func TestEventTimeExtractingUnixTimeMilliSecFormat(t *testing.T) {
	in := []byte("{\"folder\": {\"time\":1460888978777}}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "folder.time")

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	//We validate that event_time presented and has time value that equals to the input
	//(in RFC3339 format without the milliseconds part)
	eventTime, _ := gaiamap["event_time"]
	if eventTime.(string) != "2016-04-17T10:29:38Z" {
		t.Fatal("event contained the time 2016-04-17T10:29:38Z, but event_time was: ", eventTime.(string))
	}
}

func TestEventTimeExtractingBadFormat(t *testing.T) {
	in := []byte("{\"folder\": {\"time\":false}}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "folder.time")

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	//We validate that event_time presented and has time value (which is now())
	eventTime, _ := gaiamap["event_time"]
	_, err = time.Parse(time.RFC3339, eventTime.(string))
	if err != nil {
		t.Fatal("event_time:", eventTime.(string), " is not RFC3339 complient, err: ", err)
	}
}

func TestEventTimeExtractingFromArray(t *testing.T) {
	in := []byte("{\"folder\": [{\"time\":1460888978777}, {\"time\":2222222222222}]}")
	p := NewProcessor(1, "queue", "exchange")
	out, err := p.enrichMessage(&in, "folder[0].time")

	if err != nil {
		t.Fatal("err was return on passing the valid json: ", string(in), ", error is:", err)
	}

	gaiamap := extractingGaiaMap(t, out)

	//We validate that event_time presented and has time value that equals to the input
	//(in RFC3339 format without the milliseconds part)
	eventTime, _ := gaiamap["event_time"]
	if eventTime.(string) != "2016-04-17T10:29:38Z" {
		t.Fatal("event contained the time 2016-04-17T10:29:38Z, but event_time was: ", eventTime.(string))
	}
}

func extractingGaiaMap(t *testing.T, jsonData *[]byte) map[string]interface{} {

	//Unmarshal the output json and check "gaia" section exists with valid attributes
	var f interface{}
	err := json.Unmarshal(*jsonData, &f)
	if err != nil {
		t.Fatal("failed to unmarshal the outout: ", string(*jsonData), ", error is:", err)
	}

	m := f.(map[string]interface{})
	val, ok := m["gaia"]
	if !ok {
		t.Fatal("expecting the entry \"gaia\" to be presented after enrichment")
	}

	return val.(map[string]interface{})
}

//Conversion function only takes care of '.', '[' and ']'
func TestConvertFieldLocationIntoSlice(t *testing.T) {
	out := convertFieldLocationToSlice("test1.test2")
	if len(out) != 2 || out[0] != "test1" || out[1] != "test2" {
		t.Fatal("Conversion failed: ", out)
	}

	out = convertFieldLocationToSlice("test1[0]")
	if len(out) != 2 || out[0] != "test1" || out[1] != "0" {
		t.Fatal("Conversion failed: ", out)
	}

	out = convertFieldLocationToSlice("test1[0].test2")
	if len(out) != 3 || out[0] != "test1" || out[1] != "0" || out[2] != "test2" {
		t.Fatal("Conversion failed: ", out)
	}

	out = convertFieldLocationToSlice("test1[0].test2.1")
	if len(out) != 4 || out[0] != "test1" || out[1] != "0" || out[2] != "test2" || out[3] != "1" {
		t.Fatal("Conversion failed: ", out)
	}

	out = convertFieldLocationToSlice("test1[0\"!@#$%^&*(),;].test2.1")
	if len(out) != 4 || out[0] != "test1" || out[1] != "0\"!@#$%^&*(),;" || out[2] != "test2" || out[3] != "1" {
		t.Fatal("Conversion failed: ", out)
	}
}
