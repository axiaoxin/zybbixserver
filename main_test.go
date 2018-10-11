package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/json-iterator/go"
)

func TestLoadZabbixKeyMonitorItemMap(t *testing.T) {
	m := loadZabbixKeyMonitorItemMap()
	if len(m) < 1 {
		t.Error("monitems's data is empty!")
	}
	for key, item := range m {
		if key != item.ZabbixKey {
			t.Errorf("key: %s not equal to item.ZabbixKey: %s", key, item.ZabbixKey)
		}
		if item.Base <= 0 {
			t.Errorf("key:%s base=%v must > 0", key, item.Base)
		}
		if item.AttrID <= 0 {
			t.Errorf("key:%s invalid AttrID=%v", key, item.AttrID)
		}
		if item.Delay <= 0 {
			t.Errorf("key:%s delay=%v must > 0", key, item.Delay)
		}
	}
}

func TestHandleActiveChecks(t *testing.T) {
	rev := handleActiveChecks()
	header := rev[:5]
	if !bytes.Equal(header, []byte("ZBXD\x01")) {
		t.Error("error header:", header)
	}
	datalen := rev[5:13]
	t.Log("datalen:", datalen)
	int_datalen := int(binary.LittleEndian.Uint32(datalen))
	data := rev[13:]
	data_length := len(data)
	if int_datalen != data_length {
		t.Errorf("error datalen. datalen=%v, data length=%v", datalen, data_length)
	}
	any := jsoniter.Get(data)
	if any.ValueType() == jsoniter.InvalidValue {
		t.Error("data is invalid json")
	}
	if any.Get("data").Size() < 1 {
		t.Error("monitems's data is empty!")
	}
}

func TestPacketZabbixData(t *testing.T) {
	val := []byte(`{"ID":1,"Name":"Reds","Colors":["Crimson","Red","Ruby","Maroon"]}`)
	dataLen := make([]byte, 8)
	binary.LittleEndian.PutUint32(dataLen, uint32(len(val)))
	expected := append([]byte("ZBXD\x01"), dataLen...)
	expected = append(expected, val...)
	rev := packetZabbixData(val)
	if !bytes.Equal(expected, rev) {
		t.Errorf("expected: %v rev: %v", expected, rev)
	}
}
