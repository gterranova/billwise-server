package controllers

import (
	"strconv"

	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/middlewares"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func AllUsers(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	return ctx.JSON(
		models.Paginate(database.DB, &models.User{}, page),
	)
}

func GetUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	userId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	var user models.User
	err = database.DB.Preload("Role").First(&user, "id = ?", userId).Error
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(user)
}

/*
func GetUserActivities(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "activities"); err != nil {
		return err
	}
	userId, err := models.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	var activities []models.Activity
	cond := &models.Activity{UserID: userId}
	err = database.DB.Preload("Task").Find(&activities, cond).Error
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound)
	}
	return ctx.JSON(activities)
}
*/
/*
	func GetUserTasks(ctx *fiber.Ctx) error {
		if err := middlewares.IsUserAuthorized(ctx, "tasks"); err != nil {
			return err
		}
		userId, err := models.ParseID(ctx.Params("id"))
		if err != nil {
			return err
		}
		user := models.User{
			ID: userId,
		}
		err = database.DB.First(&user).Error
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound)
		}

		var tasks []models.UserTask

		totalTaskQuery := database.DB.Select("task_id, SUM(strftime('%s',hours_billed)-strftime('%s','00:00')) as totsec").Group("task_id").Table("activities")

		totalUserQuery := database.DB.Select("activities.task_id, SUM(strftime('%s',hours_billed)-strftime('%s','00:00')) as totsec").Where("user_id = ?", userId).Group("task_id").Table("activities")

		activityTotals := database.DB.Select("activities.task_id, printf('%d:%02d:00', qtotal.totsec / 3600, (qtotal.totsec % 3600) / 60) as task_hours, printf('%d:%02d:00', quser.totsec / 3600, (quser.totsec % 3600) / 60) as user_hours").
			Table("activities").
			Joins("LEFT JOIN (?) qtotal on activities.task_id = qtotal.task_id", totalTaskQuery).
			Joins("LEFT JOIN (?) quser on activities.task_id = quser.task_id", totalUserQuery).
			Group("activities.task_id")

		err = database.DB.Select("*, total.task_hours as task_hours, total.user_hours as user_hours").
			Table("tasks").
			Joins("LEFT JOIN (?) total on tasks.id = total.task_id", activityTotals).Find(&tasks).
			Error

		for i := range tasks {
			var refAmount float64
			ptype := *tasks[i].PaymentType
			switch ptype {
			case models.HourlyRate:
				refAmount = time.Duration(tasks[i].TaskHours).Hours() * (*tasks[i].PaymentAmount)
			case models.FixedFee:
				refAmount = *tasks[i].PaymentAmount
				if *tasks[i].InvoicedAmount > 0 {
					refAmount = *tasks[i].InvoicedAmount
				}
			}
			if tasks[i].UserHours == 0 || tasks[i].TaskHours == 0 {
				tasks[i].UserPerc = 0
				tasks[i].UserAmount = 0
				tasks[i].UserBaseAmount = 0
			} else {
				tasks[i].UserPerc = time.Duration(tasks[i].UserHours).Hours() / time.Duration(tasks[i].TaskHours).Hours()
				tasks[i].UserAmount = refAmount * tasks[i].UserPerc * user.Quota
				tasks[i].UserBaseAmount = tasks[i].UserAmount / time.Duration(tasks[i].UserHours).Hours()
			}

		}
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound)
		}
		return ctx.JSON(tasks)
	}
*/
func CreateUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return err
	}
	user.SetPassword(util.Config.DefaultUserPassword)
	result := database.DB.Create(&user)
	if result.Error != nil {
		return result.Error
	}
	return ctx.JSON(user)
}

func UpdateUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	userId, err := util.ParseID(ctx.Params("id"))
	if err != nil {
		return err
	}
	var user models.User
	if err := ctx.BodyParser(&user); err != nil {
		return err
	}
	user.ID = userId
	result := database.DB.Model(&user).Updates(user)
	if result.Error != nil {
		return err
	}
	return ctx.JSON(user)
}

func DeleteUser(ctx *fiber.Ctx) error {
	if err := middlewares.IsUserAuthorized(ctx, "users"); err != nil {
		return err
	}
	userId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return err
	}

	user := models.User{
		ID: userId,
	}

	database.DB.Model(&user).Association("Activities").Clear() // Remove existing many2many before deleting the role

	result := database.DB.Delete(&user)
	if result.Error != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound)
	}
	ctx.Status(fiber.StatusNoContent)
	return nil
}
