package controllers

import (
	"github.com/lithammer/dedent"
	"text/template"
)

type FilebeatInputConfigOptions struct {
	Stdout        bool
	Multiline     bool
	HostDir       string
	File          string
	Format        string
	Tags          map[string]string
	CustomConfigs map[string]string
}

type FilebeatConfigOptions struct {
	FilebeatLogLevel              string
	FilebeatMetricsEnabled        string
	FilebeatFilesRotateeverybytes string
	FilebeatMaxProcs              string
	FilebeatSetupIlmEnabled       string
}

var (
	FilebeatInputConfTemplate = template.Must(template.New("FilebeatInputConf").Parse(
		dedent.Dedent(`
{{if .Stdout}}
- type: container
{{ else }}
- type: log
{{end}}
{{if .Multiline }}
  multiline.pattern: '^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}\d*'
  multiline.negate: true
  multiline.match: after
{{end}}
  paths:
      - {{ .File }}
  scan_frequency: 1s
  fields_under_root: true
  {{if eq .Format "json"}}
  json.keys_under_root: false
  json.overwrite_keys: true
  json.add_error_key: false
  json.message_key: log
  {{end}}
  fields:
      {{range $key, $value := .Tags}}
      {{ $key }}: {{ $value }}
      {{end}}
      {{range $key, $value := .Metadata}}
      {{ $key }}: {{ $value }}
      {{end}}
  {{range $key, $value := .CustomConfigs}}
  {{ $key }}: {{ $value }}
  {{end}}
  clean_inactive: 36h
  ignore_older: 24h
  close_inactive: 2h
  close_removed: false
  clean_removed: false
  publisher_pipeline.disable_host: false
  {{range $key, $value := .Tags}}
  {{if eq $key "index"}}
  {{ $key }}: "{{ $value }}"
  {{end}}
  {{end}}
`)))

	FilebeatConfTemplate = template.Must(template.New("FilebeatConf").Parse(
		dedent.Dedent(`
path.config: /etc/filebeat
path.logs: /var/log/filebeat
path.data: /var/lib/filebeat/data
filebeat.registry.path: ${path.data}/registry
logging.level: {{ or .FilebeatLogLevel  "info" }}
logging.metrics.enabled: {{ or .FilebeatMetricsEnabled  "true" }}
logging.files.rotateeverybytes: {{ or .FilebeatFilesRotateeverybytes  "104857600" }}
max_procs: {{ or .FilebeatMaxProcs  "1" }}
setup.template.name: "filebeat"  
setup.template.pattern: "filebeat-*" 
setup.ilm.enabled: {{ or .FilebeatSetupIlmEnabled "false" }}
filebeat.config:
  modules:
    enabled: false
  inputs:
    enabled: true
    path: ${path.config}/inputs.d/*.yml
    reload.enabled: true
    reload.period: 10s
`)))
)
