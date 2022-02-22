/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/cccfs/kube-log-helper/pkg/controllers"
	"github.com/spf13/cobra"
	"log"
	"path/filepath"
)

var (
	template, base string
)
// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run this program",
	Run: func(cmd *cobra.Command, args []string) {
		baseDir, err := filepath.Abs(base)
		if err != nil {
			log.Fatal(err)
		}
		if template == "" {
			log.Fatal("template file can not be empty")
		}
		controllers.Run(template, baseDir)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&template, "template", "", "filebeat.tpl", "template filepath for filebeat.")
	runCmd.Flags().StringVarP(&base, "base", "", "/host", "directory which mount host root.")
}
