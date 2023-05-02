/*
Copyright © 2023 Gianpaolo Terranova <gianpaoloterranova@gmail.com>
*/
package cmd

import (
	"fmt"

	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"

	"github.com/spf13/cobra"
)

// permissionsCmd represents the permissions command
var permissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		database.Connect()

		roleName, permissions := args[0], args[1:]
		role := models.Role{Name: roleName}
		fmt.Printf("Adding %v to '%v' %v", permissions, roleName, role)

		if result := database.DB.Where("name = ?", roleName).Preload("Permissions").FirstOrCreate(&role); result.Error != nil {
			return result.Error
		}
		fmt.Println(role)
		for _, p := range permissions {
			found := false
			for _, currentP := range role.Permissions {
				if p == currentP.Name {
					found = true
					break
				}
			}
			if !found {
				permissionToAdd := models.Permission{Name: p}
				if result := database.DB.Where(&permissionToAdd).FirstOrCreate(&permissionToAdd); result.Error != nil {
					return result.Error
				}
				role.Permissions = append(role.Permissions, permissionToAdd)
			}
		}
		if result := database.DB.Model(&role).Updates(&role); result.Error != nil {
			return result.Error
		}
		if err := database.DB.Model(&role).Association("Permissions").Replace(role.Permissions); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(permissionsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// permissionsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// permissionsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
