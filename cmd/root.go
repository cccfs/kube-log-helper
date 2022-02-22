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
	logs "github.com/cccfs/kube-log-helper/pkg/logger"
	"github.com/kris-nova/logger"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kube-log-helper",
	Short: "kube-log-helper is kubernetes logs collection component",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			logger.Debug("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	loggerLevel := rootCmd.PersistentFlags().IntP("verbose", "v", 4, "set log level, use 0 to silence, 4 for debug")
	colorValue := rootCmd.PersistentFlags().StringP("color", "C", "true", "toggle colorized logs (valid options: true, false, fabulous)")
	cobra.OnInitialize(func() {
		logs.InitLogger(*loggerLevel, *colorValue)
	})
	rootCmd.PersistentFlags().BoolP("help", "h", false, "help for this command")
}
