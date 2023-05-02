/*
Copyright © 2023 Gianpaolo Terranova <gianpaoloterranova@gmail.com>
*/
package importCommands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
)

type AttivitaKleos struct {
	Data           string `json:"Data,omitempty"`
	Ora            string `json:"Ora,omitempty"`
	Tipo           string `json:"Tipo,omitempty"`
	Descrizione    string `json:"Descrizione,omitempty"`
	Pratica        string `json:"Pratica,omitempty"`
	Operatore      string `json:"Operatore,omitempty"`
	Tariffario     string `json:"Tariffario,omitempty"`
	Voci           string `json:"Voci,omitempty"`
	Tipo2          string `json:"Tipo2,omitempty"`
	Stato          string `json:"Stato,omitempty"`
	Tipo3          string `json:"Tipo3,omitempty"`
	DurataInMinuti string `json:"Durata (in minuti),omitempty"`
	Quantit        string `json:"Quantità,omitempty"`
	Prestazioni    string `json:"Prestazioni,omitempty"`
	Spese          string `json:"Spese,omitempty"`
	Prestazioni4   string `json:"Prestazioni4,omitempty"`
	Spese5         string `json:"Spese5,omitempty"`
}

func ImportActivities(db *gorm.DB, jsonBytes []byte) (err error) {

	// we initialize our Users array
	var kleosActivities []AttivitaKleos

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(jsonBytes, &kleosActivities)

	uid := (&uuid.UUID{}).String()
	return db.Session(&gorm.Session{NewDB: true, SkipHooks: true, DisableNestedTransaction: true}).Set("userId", uid).Transaction(func(tx *gorm.DB) error {
		activityCount := len(kleosActivities)
		for i, k := range kleosActivities {

			if k.Stato == "Non parcellabile" {
				continue
			}
			activity := models.Activity{}
			// data
			date, err := time.Parse("02/01/2006", k.Data)
			if err != nil {
				return err
			}
			activity.Date = datatypes.Date(date)

			// user
			var user models.User
			var cognome, nome, email string
			if strings.Contains(k.Operatore, " ") {
				cognome, nome, _ = strings.Cut(k.Operatore, " ")
				email = fmt.Sprintf("%v.%v@sazalex.com", strings.ToLower(nome)[0:1], strings.ToLower(cognome))
			} else {
				nome = k.Operatore
				email = fmt.Sprintf("%v@sazalex.com", strings.ToLower(nome))
			}
			if err = tx.Where(&models.User{FirstName: strings.Trim(nome, " "), LastName: cognome, Email: email}).
				Assign(&models.User{FirstName: nome, LastName: cognome, Email: email}).
				First(&user).Error; err != nil {
				user = models.User{FirstName: strings.Trim(nome, " "), LastName: cognome, Email: email}
				user.SetPassword(util.Config.DefaultUserPassword)
				if err = tx.Model(&models.Role{}).Where(&models.Role{Name: "users"}).FirstOrCreate(&user.Role).Error; err != nil {
					return err
				}
				if result := tx.Model(&models.User{}).Create(&user); result.Error != nil {
					return result.Error
				}
			}
			activity.UserID = user.ID

			// pratica
			var task models.Task
			codPratica := strings.Split(k.Pratica, " ")[0]
			if len(codPratica) == 0 {
				continue
			}
			if err := tx.Where(&models.Task{Code: codPratica}).First(&task).Error; err != nil {
				//return err
				continue
			}
			activity.TaskID = task.ID

			// tipo
			var paymentType models.PaymentType
			switch k.Tipo {
			case "Prestazione oraria":
				paymentType = models.HourlyRate
			case "Compenso Professionale":
				paymentType = models.FixedFee
			case "Spesa imponibile":
				paymentType = models.TaxableExpense
			case "Spesa esente":
				paymentType = models.TaxExemptExpense
			case "Contributo Unificato":
				paymentType = models.ContributoUnificato
			}
			activity.PaymentType = &paymentType

			// durata
			switch *activity.PaymentType {
			case models.FixedFee:
				amount, _ := strconv.ParseFloat(strings.Replace(strings.ReplaceAll(k.Prestazioni, ".", ""), ",", ".", 1), 64)
				activity.PaymentAmount = &amount
			case models.HourlyRate:
				minutes, _ := strconv.Atoi(strings.Split(k.DurataInMinuti, ",")[0])
				activity.HoursBilled = datatypes.NewTime(0, minutes, 0, 0)
			case models.TaxableExpense, models.TaxExemptExpense, models.ContributoUnificato:
				amount, _ := strconv.ParseFloat(strings.Replace(strings.ReplaceAll(k.Spese, ".", ""), ",", ".", 1), 64)
				activity.PaymentAmount = &amount
			}

			activity.Description = k.Voci

			fmt.Printf("[%v/%v] Importing %v %v %v", i+1, activityCount, k.Data, task.Name, activity.Description)
			if result := tx.Session(&gorm.Session{SkipHooks: true}).Set("userId", uid).Where(&activity).Assign(&activity).FirstOrCreate(&activity); result.Error != nil {
				fmt.Printf("--> FAILED\n")
				return result.Error
			}
			fmt.Printf("--> OK\n")
		}
		return nil
	})
}

func ImportActivitiesFromFile(db *gorm.DB, filename string) (err error) {

	if strings.HasSuffix(strings.ToUpper(filename), "JSON") {
		// Open our jsonFile
		var jsonFile *os.File
		if jsonFile, err = os.Open(filename); err != nil {
			// if we os.Open returns an error then handle it
			return err
		}
		fmt.Println("Successfully Opened " + filename)
		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()

		// read our opened jsonFile as a byte array.
		byteValue, _ := io.ReadAll(jsonFile)

		return ImportActivities(db, byteValue)
	}

	return fmt.Errorf("cannot import %v", filename)
}

// activitiesCmd represents the activities command
var activitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("import activities from", cmd.Flag("file").Value)
		filename := cmd.Flag("file").Value.String()
		database.Connect()
		return ImportActivitiesFromFile(database.DB, filename)
	},
}

func init() {
	ImportCmd.AddCommand(activitiesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// activitiesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// activitiesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
