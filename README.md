zybbixserver
============


zybbixserver 是按照 zabbix protocol 实现的自定义简易版本服务端.
支持 zabbix agent 获取监控项列表,
支持接收 zabbix agent 上报的监控数据,并将监控数据以整数ID的形式转发到其他TCP服务中.


### 开发

zybbixserver 项目使用 dep 进行包管理，开发中使用新的包请使用 dep 进行管理

### 安装并运行

确保monitems.json存在且内容正确

    ./zybbixserver

### 配置

配置项：

- bind zybbixserver 运行地址(IP:PORT) 默认 :20051
- report_to  转发到织云monitor服务地址(IP:PORT) 默认 127.0.0.1:6621
- data_path  zybbixserver 数据文件monitems.json存放目录 默认 .
- log_level  zybbixserver 日志打印级别 默认 info （可选值有： debug|info|warning|error|fatal|panic）
- log_formatter  zybbixserver 日志格式 默认 text （可选值有： text|json）
- debug  zybbixserver debug模式 默认 false

修改配置项支持三种方式：

1. 通过添加配置文件指定配置项的值，JSON、TOML、YAML、HCL 或 Java properties 格式的配置文件都可以，但是文件名必须为 `zybbixserver`，文件存放路径为程序所在目录，home 目录或者 /etc/ 下
2. 通过设置环境变量指定值，环境变量名称必须是配置项名称的大写形式且带前缀 `ZYBBIX_`
3. 运行时通过命令行参数指定

监控项数据文件：

监控项在 `monitems.json` 文件中配置，JSON格式。存放到 data_path 指定的路径下即可。

格式为：

    [
        {
            "zabbix_key": "system.cpu.util[,user]",
            "attr_id": 9,
            "delay": 60,
            "base": 1
        },
        {
            "zabbix_key": "vm.memory.size[total]",
            "attr_id": 25,
            "delay": 60,
            "base": 0.0009765625
        }
    ]

以上示例会将下发`zabbix_key`给 zabbix agent, zabbix agent 会以每隔 delay 秒进行上报监控数据，
zybbixserver接受到对应的数据后对数据按 `base` 值进行乘法运算，并将得到的值以对应的`attr_id`转发到其他TCP服务。

# Release log

2018-10-11: release v1.0


### TODO

- 数据文件变更自动reload
- 命令行参数检查配置文件和数据文件

