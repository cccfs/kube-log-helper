package controllers

const (
	FilebeatBase        string = "/etc/filebeat"
	FilebeatBin         string = "/usr/bin/filebeat"
	FilebeatConf        string = FilebeatBase + "/filebeat.yml"
	FilebeatConfDir     string = FilebeatBase + "/inputs.d"
	AlreadyStartedError string = "already started"

	EnvLoggingPath                   string = "/var/log/containers"
	EnvLoggingPrefix                 string = "LOGGING_INDEX_PREFIX" + "_logs_"
	EnvClusterEnvName                string = "CLUSTER_ENV_NAME"
	EnvFilebeatLogLevel              string = "FILEBEAT_LOG_LEVEL"
	EnvFilebeatMetricsEnabled        string = "FILEBEAT_METRICS_ENABLED"
	EnvFilebeatFilesRotateeverybytes string = "FILEBEAT_FILES_ROTATEEVERYBYTES"
	EnvFilebeatMaxProcs              string = "FILEBEAT_MAX_PROCS"
	EnvFilebeatSetupIlmEnabled       string = "FILEBEAT_SETUP_ILM_ENABLED"
)
