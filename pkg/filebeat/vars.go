package filebeat

const (
	FILEBEAT_EXEC_CMD  	= "/usr/bin/filebeat"
	FILEBEAT_REGISTRY  	= "/var/lib/filebeat/registry/filebeat"
	FILEBEAT_BASE_CONF 	= "/etc/filebeat"
	FILEBEAT_CONF_FILE 	= FILEBEAT_BASE_CONF + "/filebeat.yml"
	FILEBEAT_CONF_DIR  	= FILEBEAT_BASE_CONF + "/inputs.d"
)
