package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"git.code.oa.com/u/ashinchen/zybbixserver/lib"
	"github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const VERSION = "1.0"

type CollectorMetric [2]int

type CollectorPacket struct {
	Type string            `json:"type"`
	Mac  string            `json:"mac"`
	IP   string            `json:"ip"`
	Data []CollectorMetric `json:"data"`
}

type MonitorItem struct {
	ZabbixKey   string
	AttrID      int
	Delay       int
	LastLogSize int
	MTime       int
	Base        float64
}

type ReportResult struct {
	Processed    int
	Failed       int
	Total        int
	SecondsSpent float64
}

var ZBXHEADER = []byte("ZBXD\x01")
var ZabbixKeyMonitorItemMap = loadZabbixKeyMonitorItemMap()

func init() {
	version := flag.Bool("version", false, "show version")
	check := flag.Bool("check", false, "check everything need to be checked")
	flag.Parse()
	if *version {
		fmt.Println("zybbixserver", VERSION)
		os.Exit(0)
	}
	if *check {
		fmt.Println("zybbixserver is ok")
		os.Exit(0)
	}
	lib.InitConfig()
	lib.InitLog()
}

func main() {
	bind := viper.GetString("bind")
	tcpAddr, err := net.ResolveTCPAddr("tcp4", bind)
	if err != nil {
		log.Fatal("Resolve TCP Addr error:", err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal("Listen TCP error:", err)
	}
	log.Info("zybbixserver is running on ", bind)
	for {
		conn, err := listener.Accept()
		log.Debug("Remote addr connected:", conn.RemoteAddr())
		if err != nil {
			log.Error(err)
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()                                    // close connection before exit
	conn.SetReadDeadline(time.Now().Add(2 * time.Minute)) // 2分钟长连接
	header := make([]byte, 5)                             // zabbix request header
	_, err := conn.Read(header)
	if err != nil {
		log.Error("Read header error:", err)
	}
	if !bytes.Equal(header, ZBXHEADER) {
		log.Warn("Invalid header, ignored!")
		return
	}
	datalen := make([]byte, 8) // zabbix request data length
	_, err = conn.Read(datalen)
	if err != nil {
		log.Error("Read datalen error:", err)
	}
	datalen_int := binary.LittleEndian.Uint32(datalen)
	log.Debug("zabbix request data length:", datalen, datalen_int)
	data := make([]byte, datalen_int) // zabbix request json data
	_, err = conn.Read(data)
	if err != nil {
		log.Error("Read data error:", err)
	}
	json := jsoniter.Get(data)
	log.Debug("zabbix request json data:", json.ToString())

	request := json.Get("request").ToString()
	result := []byte("Invalid request")
	if request == "active checks" {
		result = handleActiveChecks()
	} else if request == "agent data" || request == "sender data" {
		result = handleMonitorData(json.Get("data"))
	}
	log.Debug("zybbixserver response:", string(result))
	conn.Write(result)
}

func loadZabbixKeyMonitorItemMap() map[string]MonitorItem {
	// 从JSON文件加载数据
	json, err := ioutil.ReadFile(filepath.Join(viper.GetString("data_path"), "monitems.json"))
	if err != nil {
		log.Fatal(err)
	}
	items := jsoniter.Get(json)
	zkmiMap := map[string]MonitorItem{}
	for index := 0; index < items.Size(); index++ {
		item := items.Get(index)
		mItem := MonitorItem{
			ZabbixKey:   item.Get("zabbix_key").ToString(),
			AttrID:      item.Get("attr_id").ToInt(),
			Delay:       item.Get("delay").ToInt(),
			LastLogSize: item.Get("lastlogsize").ToInt(),
			MTime:       item.Get("mtime").ToInt(),
			Base:        item.Get("base").ToFloat64(),
		}
		zkmiMap[mItem.ZabbixKey] = mItem
	}
	return zkmiMap
}

func handleActiveChecks() []byte {
	response := map[string]interface{}{
		"response": "success",
		"data":     []map[string]interface{}{},
	}
	for _, item := range ZabbixKeyMonitorItemMap {
		d := map[string]interface{}{
			"key":         item.ZabbixKey,
			"delay":       item.Delay,
			"lastlogsize": item.LastLogSize,
			"mtime":       item.MTime,
		}
		response["data"] = append(response["data"].([]map[string]interface{}), d)
	}
	json, err := jsoniter.Marshal(response)
	if err != nil {
		log.Error(err)
	}
	return packetZabbixData(json)
}

func reportToCollector(data jsoniter.Any) ReportResult {
	start := float64(time.Now().UnixNano()) / 1e6
	metrics := []CollectorMetric{}
	total := data.Size()
	processed, failed := total, 0
	result := ReportResult{
		Processed: processed,
		Failed:    failed,
		Total:     total,
	}
	for index := 0; index < total; index++ {
		zabbixDataItem := data.Get(index)
		monitorItem := ZabbixKeyMonitorItemMap[zabbixDataItem.Get("key").ToString()]
		value := int(zabbixDataItem.Get("value").ToFloat64() * monitorItem.Base)
		metrics = append(metrics, CollectorMetric{monitorItem.AttrID, value})
	}
	packet, err := jsoniter.Marshal(CollectorPacket{
		Type: "num",
		Mac:  "",
		IP:   data.Get(0).Get("host").ToString(),
		Data: metrics,
	})
	if err != nil {
		log.Error(err)
		result.Processed = 0
		result.Failed = total
	}
	log.Debug("collector packet:", string(packet))
	res, err := lib.SendTCPPacket(viper.GetString("report_to"), packet)
	if err != nil {
		log.Error("report error:", err, " response:", res)
		result.Processed = 0
		result.Failed = total
	}
	result.SecondsSpent = float64(time.Now().UnixNano())/1e6 - start
	return result
}

func handleMonitorData(data jsoniter.Any) []byte {
	// report data to collector
	reportResult := reportToCollector(data)

	// response zabbix agent
	response := map[string]string{
		"response": "success",
		"info":     fmt.Sprintf("processed: %d; failed: %d; total: %d; seconds spent: %f", reportResult.Processed, reportResult.Failed, reportResult.Total, reportResult.SecondsSpent),
	}
	json, err := jsoniter.Marshal(response)
	if err != nil {
		log.Error(err)
	}
	return packetZabbixData(json)
}

func packetZabbixData(json []byte) []byte {
	datalen := make([]byte, 8)
	binary.LittleEndian.PutUint32(datalen, uint32(len(json)))
	packet := append(ZBXHEADER, datalen...)
	packet = append(packet, json...)
	return packet
}
