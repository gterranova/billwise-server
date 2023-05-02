package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"it.terra9/billwise-server/util"
)

func UpdateUserStats(tx *gorm.DB, tasksToUpdate []uuid.UUID) (err error) {
	type TaskStats map[uuid.UUID]*TaskUserStats
	type AccountingStats map[uuid.UUID]*AccountingDocumentUserStats

	type TaskData struct {
		totalSeconds uint
		stats        TaskStats
		t            *Task
	}

	type AccountingData struct {
		totalActivitiesSeconds uint
		totalActivitiesAmounts float64
		stats                  AccountingStats
		d                      *AccountingDocument
	}

	// first initialize the tasks map
	// and fill the list of activities to update
	tasksMap := make(map[uuid.UUID]*TaskData)
	activitiesToUpdate := make([]*Activity, 0)
	for _, taskID := range tasksToUpdate {
		activityTask := GetTaskById(tx, taskID, util.SessionUserID(tx))
		tasksMap[taskID] = &TaskData{stats: make(map[uuid.UUID]*TaskUserStats), t: activityTask}
		activitiesToUpdate = append(activitiesToUpdate, activityTask.Activities...)
	}

	// iterate through the activities and calculate the totals
	// per document and per user
	accountingMap := make(map[uuid.UUID]*AccountingData)
	for _, activity := range activitiesToUpdate {
		activityTask := tasksMap[activity.TaskID].t
		activity.UpdateCalculatedFields(tx, activityTask, false)

		if activity.ReferenceDocumentID != nil {
			accountingDocumentID := activity.ReferenceDocumentID
			if activity.ReferenceDocument == nil {
				activity.ReferenceDocument = GetAccountingDocumentById(tx, *accountingDocumentID, util.SessionUserID(tx))
			}
			if _, ok := accountingMap[*accountingDocumentID]; !ok {
				accountingMap[*accountingDocumentID] = &AccountingData{
					stats: make(map[uuid.UUID]*AccountingDocumentUserStats),
					d:     activity.ReferenceDocument,
				}
			}
			if _, ok := accountingMap[*accountingDocumentID].stats[activity.UserID]; !ok {
				accountingMap[*accountingDocumentID].stats[activity.UserID] = &AccountingDocumentUserStats{
					AccountingDocumentID: *accountingDocumentID,
					UserID:               activity.UserID,
					TotalUserSeconds:     0,
					TotalUserAmounts:     0,
					UserPerc:             0,
				}
			}
			accountingMap[*accountingDocumentID].stats[activity.UserID].TotalUserSeconds += activity.SecondsBilled
			accountingMap[*accountingDocumentID].totalActivitiesSeconds += activity.SecondsBilled
			accountingMap[*accountingDocumentID].stats[activity.UserID].TotalUserAmounts += activity.CalculatedAmount
			accountingMap[*accountingDocumentID].totalActivitiesAmounts += activity.CalculatedAmount
		}
	}

	// now that we have the totals per document and user
	// calculate the users' ratio and save the user stats per
	// accounting document
	for accountingDocumentID, accountingData := range accountingMap {
		var discount float64 = 1
		if accountingMap[accountingDocumentID].totalActivitiesSeconds > 0 {
			discount = *accountingMap[accountingDocumentID].d.DocumentAmount / accountingData.totalActivitiesAmounts
			for userId := range accountingMap[accountingDocumentID].stats {
				accountingMap[accountingDocumentID].stats[userId].UserPerc = float64(accountingMap[accountingDocumentID].stats[userId].TotalUserSeconds) / float64(accountingData.totalActivitiesSeconds)
				accountingMap[accountingDocumentID].stats[userId].TotalUserAmounts *= *accountingMap[accountingDocumentID].d.DocumentAmount * accountingMap[accountingDocumentID].stats[userId].UserPerc
			}
		}
		accountingMap[accountingDocumentID].totalActivitiesAmounts *= discount

		tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()).Transaction(func(tx *gorm.DB) (err error) {
			if err = tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()).
				Exec(`UPDATE "accounting_documents" SET "total_activities_seconds"=?,"total_activities_amounts"=? WHERE "id" = ?`,
					accountingData.totalActivitiesSeconds,
					accountingData.totalActivitiesAmounts,
					accountingDocumentID).Error; err != nil {
				return
			}
			userStats := make([]*AccountingDocumentUserStats, 0)
			for _, value := range accountingData.stats {
				userStats = append(userStats, value)
			}
			return ReplaceAccountingUserStats(tx, accountingDocumentID, userStats)
		})
	}

	// iterate again through the activities and calculate the amount
	// accrued for each activity, then save the results and
	// calculate the totals per task
	for _, activity := range activitiesToUpdate {
		taskID := activity.TaskID
		// recalc activity.CalculatedAmount
		activity.UpdateCalculatedFields(tx, tasksMap[taskID].t, false)
		// save the activity's calculated fields
		if err = tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()).
			Exec(`UPDATE "activities" SET "reference_document_id"=?,"status"=?,"seconds_billed"=?,"calculated_amount"=? WHERE "id" = ?`,
				activity.ReferenceDocumentID,
				activity.Status,
				activity.SecondsBilled,
				activity.CalculatedAmount, activity.ID).Error; err != nil {
			return
		}

		if _, ok := tasksMap[taskID].stats[activity.UserID]; !ok {
			tasksMap[taskID].stats[activity.UserID] = &TaskUserStats{
				TaskID:              taskID,
				UserID:              activity.UserID,
				TotalUserSeconds:    0,
				DocumentUserBase:    0,
				DocumentUserSeconds: 0,
				PayableUserBase:     0,
				PayableUserSeconds:  0,
				PaidUserBase:        0,
				PaidUserSeconds:     0,
			}
		}
		if activity.SecondsBilled > 0 {
			tasksMap[taskID].totalSeconds += activity.SecondsBilled
			switch {
			case activity.InvoiceID != nil: // paid
				tasksMap[taskID].stats[activity.UserID].PaidUserSeconds += activity.SecondsBilled
				tasksMap[taskID].stats[activity.UserID].PaidUserBase += activity.CalculatedAmount
			case activity.Status == 0:
			case activity.Status == PaymentMade: // payable
				tasksMap[taskID].stats[activity.UserID].PayableUserSeconds += activity.SecondsBilled
				tasksMap[taskID].stats[activity.UserID].PayableUserBase += activity.CalculatedAmount
			case activity.Status < PaymentMade: // accrued
				tasksMap[taskID].stats[activity.UserID].DocumentUserSeconds += activity.SecondsBilled
				tasksMap[taskID].stats[activity.UserID].DocumentUserBase += activity.CalculatedAmount
			}

			tasksMap[taskID].stats[activity.UserID].TotalUserSeconds += activity.SecondsBilled
		}
	}

	// update the task's calculated fields
	for taskID, taskData := range tasksMap {
		err = tx.Transaction(func(tx *gorm.DB) (err error) {
			if err = tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()).
				Exec(`UPDATE "tasks" SET "total_seconds"=? WHERE "id" = ?`, taskData.totalSeconds, taskID).Error; err != nil {
				return
			}
			userStats := make([]*TaskUserStats, 0)
			for _, value := range taskData.stats {
				userStats = append(userStats, value)
			}
			return ReplaceTaskUserStats(tx.Session(&gorm.Session{NewDB: true}).Set("userId", util.SessionUserID(tx).String()), taskID, userStats)
		})
	}
	return err
}
