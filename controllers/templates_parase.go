package controllers

import "os"

func filebeatConfigParse() (string, error) {
	return Render(FilebeatConfTemplate, Data{
		"FilebeatLogLevel":              os.Getenv(EnvFilebeatLogLevel),
		"FilebeatMetricsEnabled":        os.Getenv(EnvFilebeatMetricsEnabled),
		"FilebeatFilesRotateeverybytes": os.Getenv(EnvFilebeatFilesRotateeverybytes),
		"FilebeatMaxProcs":              os.Getenv(EnvFilebeatMaxProcs),
		"FilebeatSetupIlmEnabled":       os.Getenv(EnvFilebeatSetupIlmEnabled),
	})
}

func filebeatInputConfigParse() (string, error) {

	return Render(FilebeatInputConfTemplate, Data{})
}
