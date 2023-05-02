/*
Copyright © 2023 Gianpaolo Terranova <gianpaoloterranova@gmail.com>
*/
package importCommands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ImportCmd represents the import command
var ImportCmd = &cobra.Command{
	Use:   "import",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("import called", cmd.Flag("file").Value)
	},
}

func init() {

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	ImportCmd.PersistentFlags().String("file", "", "file to import")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// importCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
