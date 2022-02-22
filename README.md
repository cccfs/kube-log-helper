# Parameters
| Environment Variable | Description | Default |
| :-----| :---- | :---- |
| LOGGING_PREFIX | the env to define prefix | k8s |
| LOGGING_OUTPUT | the services to define output | console |
| LOGGING_ENABLE_HOST | the system logs collection to define enable | true |


# Output configure
- elasticsearch
    ```yaml
    LOGGING_OUTPUT: elasticsearch
    ELASTICSEARCH_HOSTS: http:/xxx.xxx.xxx:9200
    ELASTICSEARCH_USER: kibana
    ELASTICSEARCH_PASSWORD: kibana
    ```
- kafka
    ```yaml
    LOGGING_OUTPUT: kafka
    KAFKA_BROKERS: kafka1:9092,kafka2:9092,kafka3:9092
    ```
  
# More
env变量说明:
* k8s_logs_$name=$path
  * name是生成日志索引的名称，path是日志文件路径，可以包含通配符，"stdout"是一个特殊值，表示收集容器stdout日志,遇到输出日志为'json'格式时会自动进行格式解析.
* k8s_logs_$name_tags="k1=v1,k2=v2"
* k8s_logs_$name_config="multiline.pattern='^[0-9]{3}.*',multiline.negate=true,multiline.match=after"
* k8s_logs_$name_java="true"
* k8s_logs_$name_format=none|json|csv|nginx|apache2|regexp
* k8s_logs_$name_format_pattern=$regex
* k8s_logs_$name_target="custom index name"

# Feature
---
✔ support `docker` \
~~support `system log collection`~~

# Issue

# filebeat配置说明
* close_inactive: 日志文件2小时内没有被修改将关闭harvester进程，如果日志正在采集时日志文件被删除了，此时文件句柄被filebeat保持着，文件占据的磁盘空间会被保留到harvester goroutine结束
* clean_inactive: 清理掉registry文件中距离最近一次更新超出36小时的日志文件状
* ignore_older: 忽略24小时前的日志
