#AuditLog
## 概览
对ddi资源的增、删、改的操作进行记录和存储，默认存储6个月，以满足产品的日志审计功能

## 设计
#### 日志（AuditLog）
* 顶级资源，包含字段用户名username、源IP sourceIp、操作方法 method、资源名字 resourceKind、资源URL resourcePath、 资源ID resourceId、 操作参数 parameters、操作结果succeed、错误信息 errMessage
* 支持方法：只支持获取, 支持条件搜索 
  * 源IP过滤条件：source_ip
  * 时间过滤条件：
    * 开始时间：from 格式为 2006-01-02
    * 结束时间：to 格式为 2006-01-02