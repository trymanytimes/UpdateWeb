#Alarm
## 概览
当系统硬件资源、系统状态或者业务指标超过阈值，会产生告警事件，数据库保存最近三个月告警事件，用户可以在铃铛标志处查看未处理的告警条数，可以在告警信息处查看所有告警信息

## 设计
支持的告警项

* 硬件资源
  * CPU使用率：cpuUsedRatio
  * 内存使用率：memoryUsedRatio
  * 硬盘使用率：storageUsedRatio
* 系统状态
  * 节点离线：nodeOffline
  * DNS服务离线：dnsoffline
  * DHCP服务离线：dhcpoffline
  * HA主备切换：haTrigger
* 业务指标
  * QPS: qps
  * LPS: lps
  * 地址池使用率：subnetUsedRatio
  * 地址冲突：ipConflict
  
支持的告警级别

* 紧急告警：critical
* 重要告警：major
* 次要告警：minor
* 警告告警：warning

支持的告警状态

* 未处理：untreated
* 已处理：solved
* 已忽略：ignored

## 资源
#### 发件人 （MailSender）
* 顶级资源，字段包含：用户名 username 密码 password 发件服务器 host 发件服务器端口 port 是否启用 enbaled
* 支持：增、改、查，且所有字段都支持更新

#### 收件人 （MailReceiver）
* 顶级资源，字段包含：管理员名字 name 管理员邮箱地址 address
* 支持：增、删、改、查，其中只有address支持修改

#### 阈值配置 （Threshold）
* 顶级资源，字段包含：告警项 name 告警级别 level 告警阈值 value 是否启用 enabled
* 支持：增、删、改、查，其中支持修改的字段包含：告警阈值和是否启用

#### 告警消息 （Alarm）
* 顶级资源，字段包含：告警项 name、告警级别 level、告警状态 state、节点IP nodeIp、子网 subnet、当前值 value、阈值 threshold、Master节点IP masterIp、Slave节点Ip slaveIp、HA命令haCmd、冲突IP conflictIp
* 支持：改、查，仅有state支持修改
* 查询支持条件过滤，由于告警项对应告警级别是固定的，所以告警项和告警级别不需要同时出现在过滤条件中
  * name 告警项过滤
  * level 告警级别过滤
  * state 告警状态过滤
  * 时间过滤，默认开始时间为三个月前，结束时间为当前时间
    * from 开始时间，格式为 2006-01-02
    * to 结束时间， 格式为 2006-01-02
* 当发生HA切换告警，由于情况特殊，所以不限制告警条目，其他告警都有条目限制，只有被处理或被忽略，这种类型新的告警才会被记录，具体限制如下：
  * cpu/memory/storage使用率、qps、lps、节点/dhcp/dns离线：同一种告警，每个节点只记录一条未处理的告警
  * 地址池使用率：每个节点每个子网，只记录一条未处理的告警
  * 地址冲突：每个IP，只记录一条未处理的告警
