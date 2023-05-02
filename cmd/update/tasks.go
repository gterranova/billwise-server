/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package update

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"
)

// TasksCmd represents the Tasks command
var TasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Update tasks", cmd.Flag("id").Value)
		entityId := cmd.Flag("id").Value.String()

		database.Connect()
		uid := (&uuid.UUID{}).String()
		return database.DB.Set("userId", uid).Transaction(func(tx *gorm.DB) error {

			if entityId == "" {
				var tasks []models.Task

				if result := tx.Where("archived = ?", false).Find(&tasks); result.Error == nil {
					taskCount := len(tasks)

					for i, task := range tasks {
						fmt.Printf("[%v/%v] Update %v ", i+1, taskCount, task.Name)

						if err := models.UpdateUserStats(tx, []uuid.UUID{task.ID}); err != nil {
							fmt.Println("--> FAILED")
							return err
						}
						fmt.Println("--> OK")
					}
				}
			} else {
				var task models.Task
				if result := tx.Where("id = ?", entityId).First(&task); result.Error == nil {
					fmt.Printf("[--/--] Update %v ", task.Name)

					if err := models.UpdateUserStats(tx, []uuid.UUID{task.ID}); err != nil {
						fmt.Println("--> FAILED")
						return err
					}
					fmt.Println("--> OK")

				}
			}
			return nil
		})
	},
}

func init() {
	UpdateCmd.AddCommand(TasksCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// TasksCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// TasksCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
