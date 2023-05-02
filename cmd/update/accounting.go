/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package update

import (
	"fmt"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"
)

// AccountingCmd represents the Accounting command
var AccountingCmd = &cobra.Command{
	Use:   "accounting",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Update accounting", cmd.Flag("id").Value)
		entityId := cmd.Flag("id").Value.String()

		database.Connect()
		return database.DB.Transaction(func(tx *gorm.DB) error {

			if entityId == "" {
				var accounting []models.AccountingDocument

				if result := tx.Model(&accounting).Find(&accounting); result.Error != nil {
					return result.Error
				}
				docCount := len(accounting)

				for i, doc := range accounting {
					fmt.Printf("[%v/%v] Update doc. no. %v", i+1, docCount, doc.DocumentNumber)

					//if err := doc.AfterSave(tx); err != nil {
					//	fmt.Println("--> FAILED")
					//	return err
					//}
					fmt.Println("--> OK")
				}
			} else {
				var doc models.AccountingDocument
				if result := tx.Where("id = ?", entityId).First(&doc); result.Error != nil {
					return result.Error
				}
				fmt.Printf("[--/--] Update doc. no. %v", doc.DocumentNumber)

				//if err := doc.AfterSave(tx); err != nil {
				//	fmt.Println("--> FAILED")
				//	return err
				//}
				fmt.Println("--> OK")
			}
			return nil
		})
	},
}

func init() {
	UpdateCmd.AddCommand(AccountingCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// AccountingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// AccountingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
