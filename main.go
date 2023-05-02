/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"it.terra9/billwise-server/cmd"
	"it.terra9/billwise-server/util"
)

func main() {
	util.InitEnvConfigs()
	cmd.Execute()
}
