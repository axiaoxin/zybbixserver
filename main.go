package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"

	"git.code.oa.com/u/ashinchen/zybbixserver/lib"
	"github.com/fsnotify/fsnotify"
	"github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const VERSION = "1.1"

type CollectorMetric [2]int

type CollectorPacket struct {
	Type string            `json:"type"`
	Mac  string            `json:"mac"`
	IP   string            `json:"ip"`
	Data []CollectorMetric `json:"data"`
}

type MonitorItem struct {
	AttrID      int     `json:"attr_id"`
	Delay       int     `json:"delay"`
	LastLogSize int     `json:"lastlogsize"`
	MTime       int     `json:"mtime"`
	Base        float64 `json:"base"`
}

type ReportResult struct {
	Processed    int
	Failed       int
	Total        int
	SecondsSpent float64
}

var ZBXHEADER = []byte("ZBXD\x01")
var ZabbixKeyMonitorItemMap = map[string]MonitorItem{}

func init() {
	initConfig()
	lib.InitLog()
	initZabbixKeyMonitorItemMap()
}

func initConfig() {
	default_bind := ":20051"
	default_report_to := "127.0.0.1:6621"
	default_log_level := "info"
	default_log_formatter := "text"
	default_debug := false
	default_monitems := `{"system.cpu.util[,user]":{"attr_id":9,"delay":60,"base":1}}`

	// 默认参数
	viper.SetDefault("bind", default_bind)
	viper.SetDefault("report_to", default_report_to)
	viper.SetDefault("log_level", default_log_level)
	viper.SetDefault("log_formatter", default_log_formatter)
	viper.SetDefault("debug", default_debug)
	viper.SetDefault("monitems", default_monitems)

	// 读取配置文件
	viper.SetConfigName("zybbixserver")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/")
	viper.AddConfigPath("/etc/zybbixserver")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	// 解析环境变量
	viper.SetEnvPrefix("ZYBBIX")
	viper.BindEnv("bind")
	viper.BindEnv("report_to")
	viper.BindEnv("log_level")
	viper.BindEnv("log_formatter")
	viper.BindEnv("debug")
	viper.BindEnv("monitems")

	// 解析命令行参数
	version := flag.Bool("version", false, "show version")
	check := flag.Bool("check", false, "check everything need to be checked")
	flag.String("bind", default_bind, "ZybbixServer运行地址(:PORT)")
	flag.String("report_to", default_report_to, "接收到的监控数据上报地址(IP:PORT)")
	flag.String("log_level", default_log_level, "ZybbixServer日志打印级别(debug|info|warning|error|fatal|panic)")
	flag.String("log_formatter", default_log_formatter, "ZybbixServer日志格式(json|text)")
	flag.Bool("debug", default_debug, "ZybbixServer debug模式(true|false)")
	flag.String("monitems", default_monitems, "ZybbixServer 监控项JSON数组配置")
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)
	if *version {
		fmt.Println("zybbixserver", VERSION)
		os.Exit(0)
	}
	if *check {
		fmt.Println("zybbixserver is ok")
		os.Exit(0)
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		lib.InitLog()
		initZabbixKeyMonitorItemMap()
	})
}

func initZabbixKeyMonitorItemMap() {
	// 从JSON文件加载数据
	monitems := viper.GetStringMap("monitems")
	for zabbixKey, i := range monitems {
		ZabbixKeyMonitorItemMap[zabbixKey] = MonitorItem{
			AttrID: int(i.(map[string]interface{})["attr_id"].(float64)),
			Delay:  int(i.(map[string]interface{})["delay"].(float64)),
			Base:   i.(map[string]interface{})["base"].(float64),
		}
	}
	log.Debug("init ZabbixKeyMonitorItemMap: ", ZabbixKeyMonitorItemMap)
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

func handleActiveChecks() []byte {
	response := map[string]interface{}{
		"response": "success",
		"data":     []map[string]interface{}{},
	}
	for zabbixKey, item := range ZabbixKeyMonitorItemMap {
		d := map[string]interface{}{
			"key":         zabbixKey,
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
