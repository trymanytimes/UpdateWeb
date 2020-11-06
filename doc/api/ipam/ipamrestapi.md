## ipam rest api



## 全局设置：

- 所有的变量类型都是字符串或整数
- 接口暂定，联调时候继续优化。



### 1 IPAM

#### 1.1 地址规划

##### 1.1.1 获取Plan列表
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 获取Plan列表                                                   |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans                         |
| 请求方式 | GET                                                          |
| 请求参数 | 无                                                            |

- 请求示例
```
curl 'https://10.0.0.183:58081/apis/linkingthing.com/ipam/v1/plans' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"type": "collection",
	"resourceType": "plan",
	"links": {
		"self": "/apis/linkingthing.com/ipam/v1/plans"
	},
	"data": [
		{
			"id": "b0cfad4640e257f880fdf1b0ec304d64",
			"type": "plan",
			"links": {
				"collection": "/apis/linkingthing.com/ipam/v1/plans",
				"layouts": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts",
				"remove": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64",
				"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64"
			},
			"creationTimestamp": "2020-08-14T11:14:11+08:00",
			"deletionTimestamp": null,
			"prefix": "2100::/32",
			"maskLen": 64,
			"description": ""
		}
	]
}
```

##### 1.1.2 获取Plan详情
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 获取Plan详情                                                   |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]                |
| 请求方式 | GET                                                          |
| 请求参数 | planId：string，作为url的一部分，通过获取Plan列表命令的返回值中获得   |

- 请求示例
```
curl 'https://10.0.0.183:58081/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"id": "b0cfad4640e257f880fdf1b0ec304d64",
	"type": "plan",
	"links": {
		"collection": "/apis/linkingthing.com/ipam/v1/plans",
		"layouts": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts",
		"remove": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64",
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64"
	},
	"creationTimestamp": "2020-08-14T11:14:11+08:00",
	"deletionTimestamp": null,
	"prefix": "2100::/32",
	"maskLen": 64,
	"description": ""
}
```

##### 1.1.3 创建Plan
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 创建Plan                                                      |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/                        |
| 请求方式 | POST                                                         |
| 请求参数 | 如下                                                          |

| 参数名称   | 是否必填  | 数据类型   | 备注           |
| -------- | -------- | -------- | -------------- |
| prefix   | 是       | string   | ipv6地址前缀      |
| maskLen  | 是       | int      | 掩码宽度          |
|description| 否      | string   |                 |

- 请求示例
```
curl 'https://10.0.0.183:58081/apis/linkingthing.com/ipam/v1/plans' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --data-binary '{"prefix":"2100::/32","maskLen":64,"description":""}' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"id": "b0cfad4640e257f880fdf1b0ec304d64",
	"type": "plan",
	"links": {
		"collection": "/apis/linkingthing.com/ipam/v1/plans",
		"layouts": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts",
		"remove": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64",
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64"
	},
	"creationTimestamp": "2020-08-14T11:14:11+08:00",
	"deletionTimestamp": null,
	"prefix": "2100::/32",
	"maskLen": 64,
	"description": ""
}
```


##### 1.1.4 更新Plan信息
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 更新Plan信息，目前只能更新description                            |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]                |
| 请求方式 | PUT                                                          |
| 请求参数 | planId：string，作为url的一部分，通过获取Plan列表命令的返回值中获得   |

| 参数名称       | 是否必填  | 数据类型   | 备注           |
| ------------ | -------- | -------- | -------------- |
| prefix       | 是       | string   | 作为Plan的唯一识别标识字符串，必须存在      |
| description  | 是       | string   | plan描述         |

- 请求示例
```
curl 'https://10.0.0.184:58082/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64' \
  -X 'PUT' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --data-binary '{"prefix": "1999:120::/32", "description":"plan.newname"}' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"id": "b0cfad4640e257f880fdf1b0ec304d64",
	"type": "plan",
	"links": {
		"collection": "/apis/linkingthing.com/ipam/v1/plans",
		"layouts": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts",
		"remove": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64",
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64"
	},
	"creationTimestamp": "2020-08-14T11:14:11+08:00",
	"deletionTimestamp": null,
	"prefix": "2100::/32",
	"maskLen": 64,
	"description": "plan.newname"
}
```


##### 1.1.5 获取Layout列表
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 获取某个plan下的Layout列表                                      |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]/layouts        |
| 请求方式 | GET                                                          |
| 请求参数 | planId：string，作为url的一部分                                 |

- 请求示例
```
curl 'https://10.0.0.183:58081/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"type": "collection",
	"resourceType": "layout",
	"links": {
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts"
	},
	"data": [
		{
			"id": "15bbfb4440bf5e2f8014e6baf6f49103",
			"type": "layout",
			"links": {
				"collection": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts",
		        "netnodes": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/15bbfb4440bf5e2f8014e6baf6f49103/netnodes",
				"plannedsubnets": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/15bbfb4440bf5e2f8014e6baf6f49103/plannedsubnets",
				"remove": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/15bbfb4440bf5e2f8014e6baf6f49103",
				"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/15bbfb4440bf5e2f8014e6baf6f49103",
				"update": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/15bbfb4440bf5e2f8014e6baf6f49103"
			},
			"creationTimestamp": "2020-08-21T17:19:03+08:00",
			"deletionTimestamp": null,
			"name": "testlayout",
			"nodes": null
		}
	]
}
```


##### 1.1.6 新建Layout
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 新建Layout                                                    |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]/layouts        |
| 请求方式 | POST                                                         |
| 请求参数 | planId：string，作为url的一部分。其他参数如下                     |

| 参数名称   | 是否必填  | 数据类型   | 备注           |
| -------- | -------- | -------- | -------------- |
| name     | 否       | string   | layout名称      |
| autofill | 是       | bool     | 是否是智能规划，如果为true，则由后端服务自动填充nodes里的value值，否则，nodes里的value值由前端提供        |
| firstfinished | 是       | bool     | 第一次创建是否完成        |
| nodes    | 否       | PlanNode | 节点类型         |

PlanNode节点类型定义如下：

| 参数名称   | 是否必填  | 数据类型   | 备注           |
| -------- | -------- | -------- | -------------- |
| ID       | 是       | string   | 节点唯一标识符UUID|
| pid      | 是       | string   |  父节点ID        |
| name     | 是       | string   |  节点名称        |
| prefix   | 否       | string   |  节点ipv6前缀，当autofill值为true时，由后端服务自动填充，否则，由前端输入    |
| Sequence | 是       | int      |  该节点在同级节点中的顺序编号  |
| bitWidth | 是       | int      |  该节点对应的位宽  |
| value    | 是       | string   |  该节点的取值     |
| ipv4     | 否       | string   |  该节点的ipv4地址前缀和掩码长度，可以有多个，每个之间用逗号分隔 |
| modified | 是       | int      |  该节点是否被修改过，该节点本身是新增节点或者修改过，以及有子节点被删除，取值为 1，否则为 0 |

layout的nodes list中，第一个节点一般为虚拟节点，pid必须设置为"0"，bitwidth必须为0，value必须为0.
Sequence标识该节点与其他兄弟节点之间的顺序关系；
Bitwidth标识该节点的位宽N，Value的取值在[0, 2^N - 1]区间内；
IPv4标识该节点的ipv4前缀，如有多个，用逗号分隔；
Modified标识该节点是否被修改过，用于前端调用update接口的情况。判定该节点是否被修改的依据有：
1.该节点是新增节点
2.该节点自身的值被修改
3.该节点的子节点被删除
不被判定该节点被修改的情况包括但不限于：
1.子节点被修改
2.新增子节点
准确设置Modified值，不仅有利于程序优化，也是保证逻辑正确的要求。

- 请求示例
```
curl 'https://10.0.0.184:58082/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --data-binary '{"name":"lay45", "autofill": false, "nodes":[{"id":"30","pid":"0","name":"lx","prefix": "1999:1201::/32","sequence":0,"bitWidth":0,"value":0,"modified":1},{"id":"31","pid":"30","name":"group1","prefix": "1999:1201:1::/48","sequence":1,"bitWidth":16,"value":1,"ipv4":"10.1.0.0/16,10.10.2.0/24","modified":1},{"id":"32","pid":"30","name":"group2","sequence":2,"prefix":"1999:1201:2::/48","bitWidth":16,"value":2,"ipv4":"10.2.0.0/16","modified":1},{"id":"33","pid":"31","name":"group11","1999:1201:1:1::/64",sequence":1,"bitWidth":16,"value":3,"ipv4":"10.1.1.0/24,10.12.1.0/20","modified":},{"id":"34","pid":"31","name":"group12","1999:1201:1:2::/64","sequence":2,"bitWidth":16,"value":4,"ipv4":"10.1.2.0/24,10.9.3.0/24","modified":1},{"id":"35","pid":"32","name":"group21","1999:1201:2:1::/64","sequence":1,"bitWidth":16,"value":5,"ipv4":"10.2.1.0/24,10.12.5.0/24","modified":1},{"id":"36","pid":"32","name":"group22","1999:1201:2:2::/64","sequence":2,"bitWidth":16,"value":6,"ipv4":"10.2.2.0/24,10.10.0.0/16,10.12.0.0/14","modified":1}]}' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"id": "ff0f516140f2546b802f2f95615b4253",
	"type": "layout",
	"links": {
		"collection": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts",
		"netnodes": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes",
		"plannedsubnets": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/plannedsubnets",
		"remove": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253",
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253",
		"update": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253"
	},
	"creationTimestamp": "2020-08-21T19:27:48+08:00",
	"deletionTimestamp": null,
	"name": "lay45",
    "autofill": false,
    "firstfinished": false,
	"nodes": [
		{
			"id": "30",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "0",
			"name": "lx",
			"prefix": "1999:1201::/32",
			"sequence": 0,
			"bitWidth": 0,
			"value": 0,
			"modified": 0
		},
		{
			"id": "31",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "30",
			"name": "group1",
            "prefix": "1999:1201:1::/48",
			"sequence": 1,
			"bitWidth": 16,
			"value": 1,
			"ipv4": "10.1.0.0/16,10.10.2.0/24",
			"modified": 0
		},
		{
			"id": "32",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "30",
			"name": "group2",
            "prefix": "1999:1201:2::/48",
			"sequence": 2,
			"bitWidth": 16,
			"value": 2,
			"ipv4": "10.2.0.0/16",
			"modified": 0
		},
		{
			"id": "33",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "31",
			"name": "group11",
            "prefix": "1999:1201:1:1::/64",
			"sequence": 1,
			"bitWidth": 16,
			"value": 3,
			"ipv4": "10.1.1.0/24,10.12.1.0/20",
			"modified": 0
		},
		{
			"id": "34",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "31",
			"name": "group12",
            "prefix": "1999:1201:1:2::/64",
			"sequence": 2,
			"bitWidth": 16,
			"value": 4,
			"ipv4": "10.1.2.0/24,10.9.3.0/24",
			"modified": 0
		},
		{
			"id": "35",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "32",
			"name": "group21",
            "prefix": "1999:1201:2:1::/64",
			"sequence": 1,
			"bitWidth": 16,
			"value": 5,
			"ipv4": "10.2.1.0/24,10.12.5.0/24",
			"modified": 0
		},
		{
			"id": "36",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "32",
			"name": "group22",
            "prefix": "1999:1201:2:2::/64",
			"sequence": 2,
			"bitWidth": 16,
			"value": 6,
			"ipv4": "10.2.2.0/24,10.10.0.0/16,10.12.0.0/14",
			"modified": 0
		}
	]
}
```


##### 1.1.7 获取Layout详情
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 获取Layout详情                                                 |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]/layouts/[layoutId]        |
| 请求方式 | GET                                                           |
| 请求参数 | planId：string, layoutId: string，作为url的一部分。              |

- 请求示例
```
curl 'https://10.0.0.184:58082/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"id": "ff0f516140f2546b802f2f95615b4253",
	"type": "layout",
	"links": {
		"collection": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts",
		"netnodes": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes",
		"plannedsubnets": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/plannedsubnets",
		"remove": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253",
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253",
		"update": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253"
	},
	"creationTimestamp": "2020-08-21T19:27:48+08:00",
	"deletionTimestamp": null,
	"name": "lay45",
    "autofill": false,
    "firstfinished": false,
	"nodes": [
		{
			"id": "30",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "0",
			"name": "lx",
			"prefix": "1999:1201::/32",
			"sequence": 0,
			"bitWidth": 0,
			"value": 0,
			"modified": 0
		},
		{
			"id": "31",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "30",
			"name": "group1",
            "prefix": "1999:1201:1::/48",
			"sequence": 1,
			"bitWidth": 16,
			"value": 1,
			"ipv4": "10.1.0.0/16,10.10.2.0/24",
			"modified": 0
		},
		{
			"id": "32",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "30",
			"name": "group2",
            "prefix": "1999:1201:2::/48",
			"sequence": 2,
			"bitWidth": 16,
			"value": 2,
			"ipv4": "10.2.0.0/16",
			"modified": 0
		},
		{
			"id": "33",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "31",
			"name": "group11",
            "prefix": "1999:1201:1:1::/64",
			"sequence": 1,
			"bitWidth": 16,
			"value": 3,
			"ipv4": "10.1.1.0/24,10.12.1.0/20",
			"modified": 0
		},
		{
			"id": "34",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "31",
			"name": "group12",
            "prefix": "1999:1201:1:2::/64",
			"sequence": 2,
			"bitWidth": 16,
			"value": 4,
			"ipv4": "10.1.2.0/24,10.9.3.0/24",
			"modified": 0
		},
		{
			"id": "35",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "32",
			"name": "group21",
            "prefix": "1999:1201:2:1::/64",
			"sequence": 1,
			"bitWidth": 16,
			"value": 5,
			"ipv4": "10.2.1.0/24,10.12.5.0/24",
			"modified": 0
		},
		{
			"id": "36",
			"creationTimestamp": "2020-09-02T19:23:55+08:00",
			"deletionTimestamp": null,
			"Layout": "4de306c44010e2d880c650a7ce7e1859",
			"pid": "32",
			"name": "group22",
            "prefix": "1999:1201:2:2::/64",
			"sequence": 2,
			"bitWidth": 16,
			"value": 6,
			"ipv4": "10.2.2.0/24,10.10.0.0/16,10.12.0.0/14",
			"modified": 0
		}
	]
}
```


##### 1.1.8 更新Layout
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 新建Layout                                                    |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]/layouts        |
| 请求方式 | PUT                                                          |
| 请求参数 | planId：string，作为url的一部分。其他参数如下                     |

| 参数名称   | 是否必填  | 数据类型   | 备注           |
| -------- | -------- | -------- | -------------- |
| name     | 否       | string   | layout名称      |
| autofill | 是       | bool     | 是否是智能规划，如果为true，则由后端服务自动填充nodes里的value值，否则，nodes里的value值由前端提供        |
| nodes    | 否       | PlanNode | 节点类型         |

- 请求示例
```
curl 'https://10.0.0.184:58082/apis/linkingthing.com/ipam/v1/plans/1201/layouts/766f75ae40819843802fb11579ba5d60' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --data-binary '{"creationTimestamp": "2020-09-03T19:40:01+08:00", "name":"layoutnew2", "autofill": true, "firstfinished": true,"nodes": [{"id":"30","pid":"0","name":"","sequence":0,"bitWidth":0,"value":9,"modified":1},{"id":"31","pid":"30","name":"group1new","sequence":1,"bitWidth":16,"value":7,"ipv4":"10.1.0.0/16,10.10.2.0/24","modified":1},{"id":"32","pid":"30","name":"group2new","sequence":2,"bitWidth":16,"value":8,"ipv4":"10.2.0.0/16","modified":1},{"id":"33","pid":"31","name":"group11","sequence":1,"bitWidth":16,"value":3,"ipv4":"10.1.1.0/24,10.12.2.0/20","modified":1},{"id":"34","pid":"31","name":"group12","sequence":2,"bitWidth":16,"value":7,"ipv4":"10.1.2.0/24,10.9.5.0/24","modified":1},{"id":"35","pid":"32","name":"group21","sequence":1,"bitWidth":16,"value":5,"ipv4":"10.2.1.0/24,10.12.5.0/24","modified":1},{"id":"36","pid":"32","name":"group22","sequence":2,"bitWidth":16,"value":6,"ipv4":"10.2.2.0/24,10.10.0.0/16,10.12.0.0/14","modified":1}]}' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"id": "766f75ae40819843802fb11579ba5d60",
	"type": "layout",
	"links": {
		"collection": "/apis/linkingthing.com/ipam/v1/plans/1201/layouts",
		"netnodes": "/apis/linkingthing.com/ipam/v1/plans/1201/layouts/766f75ae40819843802fb11579ba5d60/netnodes",
		"plannedsubnets": "/apis/linkingthing.com/ipam/v1/plans/1201/layouts/766f75ae40819843802fb11579ba5d60/plannedsubnets",
		"remove": "/apis/linkingthing.com/ipam/v1/plans/1201/layouts/766f75ae40819843802fb11579ba5d60",
		"self": "/apis/linkingthing.com/ipam/v1/plans/1201/layouts/766f75ae40819843802fb11579ba5d60",
		"update": "/apis/linkingthing.com/ipam/v1/plans/1201/layouts/766f75ae40819843802fb11579ba5d60"
	},
	"creationTimestamp": "2020-09-08T10:16:11+08:00",
	"deletionTimestamp": null,
	"name": "layoutnew2",
	"autofill": true,
    "firstfinished": true,
	"nodes": [
		{
			"id": "30",
			"creationTimestamp": "2020-09-08T10:16:11+08:00",
			"deletionTimestamp": null,
			"Layout": "766f75ae40819843802fb11579ba5d60",
			"pid": "0",
			"name": "",
			"prefix": "1999:1201::/32",
			"sequence": 0,
			"bitWidth": 0,
			"value": 9,
			"modified": 0
		},
		{
			"id": "31",
			"creationTimestamp": "2020-09-08T10:16:11+08:00",
			"deletionTimestamp": null,
			"Layout": "766f75ae40819843802fb11579ba5d60",
			"pid": "30",
			"name": "group1new",
			"prefix": "1999:1201:1::/48",
			"sequence": 1,
			"bitWidth": 16,
			"value": 1,
			"ipv4": "10.1.0.0/16,10.10.2.0/24",
			"modified": 0
		},
		{
			"id": "32",
			"creationTimestamp": "2020-09-08T10:16:11+08:00",
			"deletionTimestamp": null,
			"Layout": "766f75ae40819843802fb11579ba5d60",
			"pid": "30",
			"name": "group2new",
			"prefix": "1999:1201:2::/48",
			"sequence": 2,
			"bitWidth": 16,
			"value": 2,
			"ipv4": "10.2.0.0/16",
			"modified": 0
		},
		{
			"id": "33",
			"creationTimestamp": "2020-09-08T10:16:11+08:00",
			"deletionTimestamp": null,
			"Layout": "766f75ae40819843802fb11579ba5d60",
			"pid": "31",
			"name": "group11",
			"prefix": "1999:1201:1:1::/64",
			"sequence": 1,
			"bitWidth": 16,
			"value": 1,
			"ipv4": "10.1.1.0/24,10.12.2.0/20",
			"modified": 0
		},
		{
			"id": "34",
			"creationTimestamp": "2020-09-08T10:16:11+08:00",
			"deletionTimestamp": null,
			"Layout": "766f75ae40819843802fb11579ba5d60",
			"pid": "31",
			"name": "group12",
			"prefix": "1999:1201:1:2::/64",
			"sequence": 2,
			"bitWidth": 16,
			"value": 2,
			"ipv4": "10.1.2.0/24,10.9.5.0/24",
			"modified": 0
		},
		{
			"id": "35",
			"creationTimestamp": "2020-09-08T10:16:11+08:00",
			"deletionTimestamp": null,
			"Layout": "766f75ae40819843802fb11579ba5d60",
			"pid": "32",
			"name": "group21",
			"prefix": "1999:1201:2:1::/64",
			"sequence": 1,
			"bitWidth": 16,
			"value": 1,
			"ipv4": "10.2.1.0/24,10.12.5.0/24",
			"modified": 0
		},
		{
			"id": "36",
			"creationTimestamp": "2020-09-08T10:16:11+08:00",
			"deletionTimestamp": null,
			"Layout": "766f75ae40819843802fb11579ba5d60",
			"pid": "32",
			"name": "group22",
			"prefix": "1999:1201:2:2::/64",
			"sequence": 2,
			"bitWidth": 16,
			"value": 2,
			"ipv4": "10.2.2.0/24,10.10.0.0/16,10.12.0.0/14",
			"modified": 0
		}
	]
}
```


##### 1.1.9 获取ipv6网络list详情
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 获取ipv6网络list详情                                            |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]/layouts/[layoutId]/netnodes?nettype=netv6        |
| 请求方式 | GET                                                           |
| 请求参数 | planId：string, layoutId: string，作为url的一部分;nettype后跟参数netv6。              |
| 返回数据 | NetNode              |

NetNode节点类型定义如下：

| 参数名称   | 是否必填  | 数据类型   | 备注           |
| -------- | -------- | -------- | -------------- |
| id       | 是       | string   | NetNode ID      |
| NetItems | 是       | []NetItem  | NetItem数组，每个NetItem对应layout里的一个PlanNode节点        |

NetItem节点类型定义如下：

| 参数名称   | 是否必填  | 数据类型   | 备注           |
| -------- | -------- | -------- | -------------- |
| Name      | 是       | string   | 节点名，与对应的PlanNode节点名相同|
| Prefix    | 是       | string   | 该节点的网络IP地址及掩码，比如：“1999:120:1::/48”     |
| Tags      | 是       | string   | 该节点的标识字符串，以 “[plan].description,[layout].name,[parent_node.Name],Name”的格式表示        |
| Level     | 是       | string   | 该节点的层级表示，如："1.2"        |
| Usage     | 是       | string   | 该节点的使用率，以当前节点为父节点，其已规划子节点数与所有可能的子节点数的比值        |

- 请求示例
```
curl 'https://10.0.0.184:58082/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes?nettype=netv6' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"type": "collection",
	"resourceType": "netnode",
	"links": {
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes"
	},
	"data": [
		{
			"type": "netnode",
			"links": {
				"collection": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes"
			},
			"creationTimestamp": null,
			"deletionTimestamp": null,
			"netitems": [
				{
					"name": "group1",
					"prefix": "1999:120:1::/48",
					"tags": "planA,layout1,group1",
					"level": "1",
					"usage": "0%"
				},
				{
					"name": "group11",
					"prefix": "1999:120:1:3::/64",
					"tags": "planA,layout1,group1,group11",
					"level": "1.1",
					"usage": ""
				},
				{
					"name": "group12new",
					"prefix": "1999:120:1:4::/64",
					"tags": "planA,layout1,group1,group12new",
					"level": "1.2",
					"usage": ""
				},
				{
					"name": "group2",
					"prefix": "1999:120:2::/48",
					"tags": "planA,layout1,group2,group12new",
					"level": "2",
					"usage": "0%"
				},
				{
					"name": "group21",
					"prefix": "1999:120:2:5::/64",
					"tags": "planA,layout1,group2,group21",
					"level": "2.1",
					"usage": ""
				},
				{
					"name": "group22",
					"prefix": "1999:120:2:6::/64",
					"tags": "planA,layout1,group2,group22",
					"level": "2.2",
					"usage": ""
				}
			]
		}
	]
}
```


##### 1.1.10 获取ipv4网络list详情
| 功能     | 描述                                                         |
| -------- | ----------------------------------------------------------- |
| 接口功能 | 获取ipv4网络list详情                                            |
| 接口地址 | /apis/linkingthing.com/ipam/v1/plans/[planId]/layouts/[layoutId]/netnodes?nettype=netv4        |
| 请求方式 | GET                                                           |
| 请求参数 | planId：string, layoutId: string，作为url的一部分;nettype后跟参数netv4。              |
| 返回数据 | NetNode              |

- 请求示例
```
curl 'https://10.0.0.184:58082/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes?nettype=netv4' \
  -H 'authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTc0NjEwNDAsImlzcyI6Imx0IGRkaSB1c2VyIGxvZ2luIn0.JIwuj3zuB8_c0Kv5xB2fZs6GUyOl5Yl21X4R1E_BRNM' \
  --compressed \
  --insecure
```
- 返回数据示例
```
{
	"type": "collection",
	"resourceType": "netnode",
	"links": {
		"self": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes"
	},
	"data": [
		{
			"type": "netnode",
			"links": {
				"collection": "/apis/linkingthing.com/ipam/v1/plans/b0cfad4640e257f880fdf1b0ec304d64/layouts/ff0f516140f2546b802f2f95615b4253/netnodes"
			},
			"creationTimestamp": null,
			"deletionTimestamp": null,
			"netitems": [
				{
					"name": "group1",
					"prefix": "10.1.0.0/16",
					"tags": "planA,layout1,group1",
					"level": "1",
					"usage": "0%"
				},
				{
					"name": "group11",
					"prefix": "10.1.1.0/24",
					"tags": "planA,layout1,group1,group11",
					"level": "1.1",
					"usage": "0%"
				},
				{
					"name": "group12",
					"prefix": "10.1.2.0/24",
					"tags": "planA,layout1,group1,group12",
					"level": "1.2",
					"usage": "0%"
				},
				{
					"name": "group2",
					"prefix": "10.2.0.0/16",
					"tags": "planA,layout1,group2",
					"level": "2",
					"usage": "0%"
				},
				{
					"name": "group21",
					"prefix": "10.2.1.0/24",
					"tags": "planA,layout1,group2,group21",
					"level": "2.1",
					"usage": "0%"
				},
				{
					"name": "group22",
					"prefix": "10.2.2.0/24",
					"tags": "planA,layout1,group2,group22",
					"level": "2.2",
					"usage": "0%"
				},
				{
					"name": "group12",
					"prefix": "10.9.3.0/24",
					"tags": "planA,layout1,group12",
					"level": "3",
					"usage": "0%"
				},
				{
					"name": "group22",
					"prefix": "10.10.0.0/16",
					"tags": "planA,layout1,group22",
					"level": "4",
					"usage": "0%"
				},
				{
					"name": "group1",
					"prefix": "10.10.2.0/24",
					"tags": "planA,layout1,group22,group1",
					"level": "4.1",
					"usage": "0%"
				},
				{
					"name": "group22",
					"prefix": "10.12.0.0/14",
					"tags": "planA,layout1,group22",
					"level": "5",
					"usage": "0%"
				},
				{
					"name": "group11",
					"prefix": "10.12.0.0/20",
					"tags": "planA,layout1,group22,group11",
					"level": "5.1",
					"usage": "0%"
				},
				{
					"name": "group21",
					"prefix": "10.12.5.0/24",
					"tags": "planA,layout1,group22,group11,group21",
					"level": "5.1.1",
					"usage": "0%"
				}
			]
		}
	]
}
```

