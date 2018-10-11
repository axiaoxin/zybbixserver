package lib

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func InitConfig() {
	default_bind := ":20051"
	default_report_to := "127.0.0.1:6621"
	default_data_path := "."
	default_log_level := "info"
	default_log_formatter := "text"
	default_debug := false

	// 默认参数
	viper.SetDefault("bind", default_bind)
	viper.SetDefault("report_to", default_report_to)
	viper.SetDefault("data_path", default_data_path)
	viper.SetDefault("log_level", default_log_level)
	viper.SetDefault("log_formatter", default_log_formatter)
	viper.SetDefault("debug", default_debug)

	// 读取配置文件
	viper.SetConfigName("zybbixserver")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/")
	viper.AddConfigPath("/etc/")
	err := viper.ReadInConfig()
	if err != nil {
		log.Warn(err)
	}

	// 解析环境变量
	viper.SetEnvPrefix("ZYBBIX")
	viper.BindEnv("bind")
	viper.BindEnv("report_to")
	viper.BindEnv("data_path")
	viper.BindEnv("log_level")
	viper.BindEnv("log_formatter")
	viper.BindEnv("debug")

	// 解析命令行参数
	flag.String("bind", default_bind, "ZybbixServer运行地址(:PORT)")
	flag.String("report_to", default_report_to, "接收到的监控数据上报地址(IP:PORT)")
	flag.String("data_path", default_data_path, "ZybbixServer数据文件存放路径")
	flag.String("log_level", default_log_level, "ZybbixServer日志打印级别(debug|info|warning|error|fatal|panic)")
	flag.String("log_formatter", default_log_formatter, "ZybbixServer日志格式(json|text)")
	flag.Bool("debug", default_debug, "ZybbixServer debug模式(true|false)")
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		InitLog()
	})
}
