/*
Copyright © 2023 Gianpaolo Terranova <gianpaoloterranova@gmail.com>
*/
package importCommands

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"
	"it.terra9/billwise-server/util"

	"github.com/google/uuid"
	"github.com/nguyenthenguyen/docx"
	"github.com/spf13/cobra"
)

type DocumentList []struct {
	Intestatario      string `json:"Intestatario,omitempty"`
	Tipo              string `json:"Tipo,omitempty"`
	NomePratica       string `json:"Nome pratica,omitempty"`
	Data              string `json:"Data,omitempty"`
	Numero            string `json:"Numero,omitempty"`
	TotaleNetto       string `json:"Totale Netto,omitempty"`
	InviaPer          string `json:"Invia per,omitempty"`
	Incasso           string `json:"Incasso,omitempty"`
	Nota              string `json:"Nota,omitempty"`
	StatoIncasso      string `json:"Stato incasso,omitempty"`
	StatoFatturazione string `json:"Stato Fatturazione,omitempty"`
	DataScadenza      string `json:"Data scadenza,omitempty"`
	TotaleImponibile  string `json:"Totale Imponibile,omitempty"`
}

type ParcellazioneKleos struct {
	Date           string `json:"date,omitempty"`
	DocumentType   string `json:"document_type,omitempty"`
	DocumentNumber string `json:"document_number,omitempty"`
	TaskName       string `json:"task_name,omitempty"`
	Activities     []struct {
		Date          string             `json:"date,omitempty"`
		UserName      string             `json:"user_name,omitempty"`
		Description   string             `json:"description,omitempty"`
		PaymentType   models.PaymentType `json:"payment_type,omitempty"`
		PaymentAmount string             `json:"payment_amount,omitempty"`
		HoursBilled   string             `json:"hours_billed,omitempty"`
	} `json:"activities,omitempty"`
	DocumentAmount string `json:"document_amount,omitempty"`
}

func ImportAccountingDocumentList(db *gorm.DB, jsonBytes []byte) (err error) {
	var documentList DocumentList

	if err = json.Unmarshal(jsonBytes, &documentList); err != nil {
		return err
	}
	uid := (&uuid.UUID{}).String()
	return db.Session(&gorm.Session{NewDB: true, SkipHooks: true, DisableNestedTransaction: true}).Set("userId", uid).Transaction(func(tx *gorm.DB) error {
		documentCount := len(documentList)
		for i, k := range documentList {

			accountingDocument := models.AccountingDocument{}
			switch k.Tipo {
			case "Fattura":
				accountingDocument.DocumentType = models.InvoiceType
			case "Proforma":
				accountingDocument.DocumentType = models.ProformaType
			default:
				continue
			}
			// data
			date, err := time.Parse("02/01/2006", k.Data[:10])
			if err != nil {
				return err
			}
			accountingDocument.Date = datatypes.Date(date)
			if number, err := strconv.ParseInt(k.Numero, 10, 32); err == nil {
				accountingDocument.DocumentNumber = int(number)
			}

			if k.StatoIncasso != "-" {
				accountingDocument.Status = k.StatoIncasso
			} else {
				accountingDocument.Status = k.StatoFatturazione
			}
			imponibile := float64(0)
			if imponibile, err = strconv.ParseFloat(strings.Replace(strings.ReplaceAll(k.TotaleImponibile, ".", ""), ",", ".", 1), 64); err == nil {
				accountingDocument.DocumentAmount = &imponibile
			}
			// pratica
			var task models.Task
			if len(k.NomePratica) == 0 {
				continue
			}
			if err := tx.Where(&models.Task{Name: k.NomePratica}).First(&task).Error; err != nil {
				//return err
				continue
			}
			accountingDocument.Task = task
			fmt.Printf("[%v/%v] Importing %v %v %v %v", i+1, documentCount, k.Data, task.Name, k.Tipo, k.Numero)
			if result := tx.Session(&gorm.Session{SkipHooks: true}).Set("userId", uid).Where(&accountingDocument).Assign(&accountingDocument).FirstOrCreate(&accountingDocument); result.Error != nil {
				fmt.Printf("--> FAILED\n")
				return result.Error
			}
			fmt.Printf("--> OK\n")
		}
		return nil
	})
}

func ImportAccountingDocument(db *gorm.DB, jsonBytes []byte) (err error) {
	var kleosDocument ParcellazioneKleos

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	if err = json.Unmarshal(jsonBytes, &kleosDocument); err != nil {
		return err
	}
	fmt.Printf("[*] %v n. %v of %v\n", kleosDocument.DocumentType, kleosDocument.DocumentNumber, kleosDocument.Date)
	date, err := time.Parse("02/01/2006", kleosDocument.Date)
	if err != nil {
		return err
	}

	var documentType models.DocumentType
	switch kleosDocument.DocumentType {
	case "Fattura":
		documentType = models.InvoiceType
	case "Proforma":
		documentType = models.ProformaType
	default:
		return fmt.Errorf("cannot import %v", kleosDocument.DocumentType)
	}
	documentNumber, _ := strconv.Atoi(kleosDocument.DocumentNumber)

	uid := (&uuid.UUID{}).String()
	return db.Session(&gorm.Session{NewDB: true, SkipHooks: true, DisableNestedTransaction: true}).Set("userId", uid).Transaction(func(tx *gorm.DB) error {

		task := models.Task{}
		if result := tx.Where("name = ?", kleosDocument.TaskName).First(&task); result.Error != nil {
			return result.Error
		}

		documentAmount, _ := strconv.ParseFloat(strings.Replace(strings.ReplaceAll(kleosDocument.DocumentAmount, ".", ""), ",", ".", 1), 64)

		accountingDocument := models.AccountingDocument{
			Date:           datatypes.Date(date),
			DocumentType:   documentType,
			DocumentNumber: documentNumber,
			TaskID:         task.ID,
			DocumentAmount: &documentAmount,
		}
		if result := tx.Where(&models.AccountingDocument{
			Date:           accountingDocument.Date,
			DocumentType:   documentType,
			DocumentNumber: documentNumber}).Assign(&accountingDocument).FirstOrCreate(&accountingDocument); result.Error != nil {
			return result.Error
		}

		documentActivities := make([]*models.Activity, 0)
		// we iterate through every user within our users array and
		// print out the user Type, their name, and their facebook url
		// as just an example

		for i, k := range kleosDocument.Activities {
			if k.UserName == "" {
				continue
			}

			activity := models.Activity{}

			activity.PaymentType = &k.PaymentType

			// durata
			switch *activity.PaymentType {
			case models.FixedFee:
				amount, _ := strconv.ParseFloat(strings.Replace(strings.ReplaceAll(k.PaymentAmount, ".", ""), ",", ".", 1), 64)
				activity.PaymentAmount = &amount
			case models.HourlyRate:
				duration, err := ParseDuration(k.HoursBilled)
				if err != nil {
					activity.HoursBilled = datatypes.NewTime(0, int(duration.Minutes()), 0, 0)
				}
			case models.TaxableExpense, models.TaxExemptExpense, models.ContributoUnificato:
				// other types are not exported anyway
				continue
			}

			// data
			date, err := time.Parse("02/01/2006", k.Date)
			if err != nil {
				return err
			}
			activity.Date = datatypes.Date(date)

			// user
			var user models.User
			var cognome, nome, email string
			if strings.Contains(k.UserName, " ") {
				cognome, nome, _ = strings.Cut(k.UserName, " ")
				email = fmt.Sprintf("%v.%v@sazalex.com", strings.ToLower(nome)[0:1], strings.ToLower(cognome))
			} else {
				nome = k.UserName
				email = fmt.Sprintf("%v@%v", strings.ToLower(nome), util.Config.DefaultUserDomain)
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
			activity.TaskID = task.ID

			if _, description, found := strings.Cut(k.Description, " - "); found {
				activity.Description = description
			} else {
				activity.Description = k.Description
			}
			fmt.Printf("[%v/%v] %v %v %v", i+1, len(kleosDocument.Activities)-2, k.Date, task.Name, activity.Description)
			if result := tx.Session(&gorm.Session{NewDB: true, SkipHooks: true, DisableNestedTransaction: true}).Set("userId", uid).Where(&models.Activity{Date: activity.Date, UserID: user.ID, TaskID: task.ID, Description: activity.Description}).Assign(&activity).FirstOrCreate(&activity); result.Error != nil {
				fmt.Printf(" FAILED\n")
				return result.Error
			}
			fmt.Printf(" OK\n")
			documentActivities = append(documentActivities, &activity)
		}

		if err := models.ReplaceAccountingDocumentActivities(tx, accountingDocument.ID, documentActivities); err != nil {
			return err
		}

		return models.UpdateUserStats(tx, []uuid.UUID{task.ID})
	})
}

func ImportAccountingDocumentFromFile(db *gorm.DB, filename string) (err error) {

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

		return ImportAccountingDocumentList(db, byteValue)
	}

	var r *docx.ReplaceDocx
	if r, err = docx.ReadDocxFile(filename); err != nil {
		return err
	}
	defer r.Close()
	m := regexp.MustCompile("<[^>]*>")
	res := m.ReplaceAllString(r.Editable().GetContent(), "")
	//fmt.Println(html.UnescapeString(res))
	return ImportAccountingDocument(db, []byte(html.UnescapeString(res)))

}

// accountingCmd represents the accounting command
var accountingCmd = &cobra.Command{
	Use:   "accounting",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("import accounting from", cmd.Flag("file").Value)
		filename := cmd.Flag("file").Value.String()
		database.Connect()

		err := filepath.Walk(filename, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || filename[0] == '~' {
				// do nothing for directories
				return nil
			}
			return ImportAccountingDocumentFromFile(database.DB, path)
		})

		if err != nil {
			fmt.Println(err)
		}

		return err
	},
}

func init() {
	ImportCmd.AddCommand(accountingCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// accountingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// accountingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func ParseDuration(st string) (time.Duration, error) {
	var h, m int
	n, err := fmt.Sscanf(st, "%d:%d", &h, &m)
	if err != nil || n != 2 {
		return 0, err
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, nil
}
