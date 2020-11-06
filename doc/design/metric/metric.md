# Metric
## 概要
通过ddi-agent暴露dns和dhcp的业务指标到prometheus，ddi-controller从proemtheus取dns和dhcp的业务指标返回给请求端

## 动机和目的
实时监测dns和dhcp的业务指标，方便用户查看业务行为和状态

## 资源
### Dns
* 父资源为Node
* 字段：Qps、CacheHit、ResolvedRatios、QueryTypeRatios、TopTenIps、TopTenDomains
* 支持操作
  * 获取（Get、List）
    * List 会返回Dns数组，每个成员根据返回的成员ID表示哪个字段有效
  * 导出csv的Action（action=exportcsv）
    * Input: 
      * period 类型为int，有效值为6、12、24、168、720、2160
    * Output：
      * path: 类型为string，表示文件存储的绝对路径

* 每个字段介绍如下
  * 每秒请求个数(Qps)
    * 从prometheus中获取一段时间内，各个时间点的qps值
    * 字段：带时间戳的qps值

			type Qps struct ｛
				Values	[]ValueWithTimestamp `json:"values"`
			｝
		
			type ValueWithTimestamp struct ｛
				Timestamp restresource.ISOTime `json:"timestamp"`
				Value     uint64 				`json:"value"`
			｝

  * 缓存命中(CacheHit)
    * 从prometheus中获取一段时间内，各个时间点的缓存命中值
    * 字段：带时间戳的缓存命中数，具体结构与qps相同

  * 请求源Top 10 IP(TopTenIps)
    * 从elasticsearch中获取一段时间内的top 10 请求源地址的值，即返回TopIp的数组
    * 字段：请求源地址 ip和 请求源地址的个数 count

			type TopIp struct {
    			Ip     string `json:"ip"`
    			Count  uint64 `json:"count"`
			}
		
  * 请求源Top 10 Domain(TopTenDomains)
    * 从elasticsearch中获取一段时间内的top 10 请求源的域名的值，即返回TopDomain的数组
    * 字段：请求源的域名 domain和 请求源域名的个数 count

			type TopDomain struct {
    			Domain	string `json:"domain"`
    			Count   uint64 `json:"count"`
			}
		
  * 解析类型比率(QueryTypeRatios)
    * 从prometheus中获取一段时间内，各个时间点的每个请求类型的比率，即返回QueryTypeRatio数组
    * 字段：包含需要解析的类型QueryType 和 带时间戳的占比率

			type QueryTypeRatio struct {
				QueryType	string         			`json:"queryType"`
				Ratios     []RatioWithTimestamp 	`json:"ratios"`
			}
			type RatioWithTimestamp struct {
				Timestamp restresource.ISOTime `json:"timestamp"`
				Ratio string               		`json:"ratio"`
			}
		

  * 解析成功率(ResolvedRatios) 
    * 从prometheus中获取一段时间内，各个时间点的解析率，即返回QueryTypeRatio数组
    * 支持的Rcode为：NOERROR、SERVFAIL、REFUSED、NXDOMAIN
    * 成功率只关注返回码 Rcode 为 NOERROR的值
    * 字段：解析返回值Rcode 和 带时间戳的各种类型的比率

			type QueryTypeRatio struct {
				Rcode	string         			`json:"rcode"`
				Ratios []RatioWithTimestamp 	`json:"ratios"`
			}
		
### Dhcp
* 父资源为Node
* 字段包含: Lps、Lease、Packets、SubnetUsedRatios
* 只支持获取（List、Get）和Action（action=exportcsv），方式与字段和Dns一致
* 每个字段介绍如下
  * 每秒租赁个数(Lps)
    * 从prometheus中获取一段时间内各个时间点的lps值
    * 字段：带时间戳的lps值，结构类型与qps相同

  * 租赁总数(Lease)
    * 从prometheus中获取一段时间内，各个时间点的租赁总数
    * 字段：带时间戳的lease总数的值，结构类型与qps相同

  * 报文统计(Packets)
    * 从prometheus中获取一段时间内，各个时间点每个报文的总数，即返回Packet数组
    * 支持的报文：discover、offer、request、ack
    * 字段：报文类型 PacketType 和 带时间戳的总数值 

			type Packet struct {
    			PacketType	string               `json:"packetType"`
    			Values      []ValueWithTimestamp `json:"values"`
			}
		
  * 地址池使用率(SubnetUsedRatios)
    * 从prometheus中获取一段时间内，各个时间点每个子网的使用率，即返回SubnetUsedRatio数组
    * 字段：子网地址Ipnet和带时间戳的使用率

			type SubnetUsedRatio struct {
    			Ipnet      string               `json:"ipnet"` 
    			UsedRatios []RatioWithTimestamp `json:"usedRatios"`
			}


