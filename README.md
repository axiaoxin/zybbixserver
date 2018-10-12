zybbixserver
============


zybbixserver 是按照 zabbix protocol 实现的自定义简易版本服务端.
支持 zabbix agent 获取监控项列表,
支持接收 zabbix agent 上报的监控数据,并将监控数据以整数ID的形式转发到其他TCP服务中.


### 开发

zybbixserver 项目使用 dep 进行包管理，开发中使用新的包请使用 dep 进行管理

### 安装运行

将 zybbixserver.tar.gz 解压即可 **直接运行**

    tar xzf zybbixserver.tar.gz -C /usr/local/
    cd /usr/local/zybbixserver
    mkdir /data/log/
    ./zybbixserver  # 前台运行
    (./zybbixserver >> /data/log/zybbixserver.log &) # 后台运行

如果使用 supervisor 进行管理： 将目录中的 supervisor.conf 文件添加到 supervisor 对应的配置文件路径

### 配置

**配置项：**

- bind zybbixserver 运行地址(IP:PORT) 默认 :20051
- report_to  转发到织云monitor服务地址(IP:PORT) 默认 127.0.0.1:6621
- log_level  zybbixserver 日志打印级别 默认 info （可选值有： debug|info|warning|error|fatal|panic）
- log_formatter  zybbixserver 日志格式 默认 text （可选值有： text|json）
- debug  zybbixserver debug模式 默认 false
- monitems  监控项JSON对象 key为zabbix key，值为JSON对象，包含`attr_id`,`delay`,`base`字段

**修改配置项支持三种方式：**

1. 通过配置文件指定配置项的值，JSON、TOML、YAML、HCL 或 Java properties 格式的配置文件都可以，但是文件名必须为 `zybbixserver`，文件存放路径为程序所在目录，home 目录或者 /etc/zybbixserver/ 下
2. 通过设置环境变量指定值，环境变量名称必须是配置项名称的大写形式且带前缀 `ZYBBIX_`
3. 运行时通过命令行参数指定

**监控项JSON对象：**

zybbixserver 会下发 zybbixserver.json 中 monitems 中的 key 给 zabbix agent 作为监控项上报, zabbix agent 会以每隔 `delay` 秒进行上报监控数据，
zybbixserver 接收到数据后对数据按 `base` 值进行乘法运算，并将得到的值以对应的 `attr_id` 转发到其他TCP服务。

# Release log

2018-10-12: release v1.1

    将 monitems.json 数据文件合并到 zybbixserver.json 中，去除 data_path 配置项
    支持对 monitems 修改后免重启即可更新监控项数据

2018-10-11: release v1.0
