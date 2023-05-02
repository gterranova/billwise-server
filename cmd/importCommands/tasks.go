/*
Copyright © 2023 Gianpaolo Terranova <gianpaoloterranova@gmail.com>
*/
package importCommands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/models"

	"github.com/spf13/cobra"
)

type PraticaKleos struct {
	ID                    string `json:"ID,omitempty"`
	PosizioneArchivio     string `json:"Posizione archivio,omitempty"`
	NomePratica           string `json:"Nome Pratica,omitempty"`
	DataCreazione         string `json:"Data creazione,omitempty"`
	DataEvento            string `json:"Data evento,omitempty"`
	Descrizione           string `json:"Descrizione,omitempty"`
	Tipo                  string `json:"Tipo,omitempty"`
	Lingua                string `json:"Lingua,omitempty"`
	DataDiArchiviazione   string `json:"Data di archiviazione,omitempty"`
	NumeroDiArchiviazione string `json:"Numero di archiviazione,omitempty"`
	InCaricoA             string `json:"In carico a,omitempty"`
	DataIncarico          string `json:"Data incarico,omitempty"`
	MotivoIncarico        string `json:"Motivo incarico,omitempty"`
	Note                  string `json:"Note,omitempty"`
	Materia               string `json:"Materia,omitempty"`
	Cliente               string `json:"Cliente,omitempty"`
	Titolare              string `json:"Titolare,omitempty"`
	Controparte           string `json:"Controparte,omitempty"`
	Affidatario           string `json:"Affidatario,omitempty"`
	AltraParte            string `json:"Altra Parte,omitempty"`
	Consulente            string `json:"Consulente,omitempty"`
}

func ImportTasks(db *gorm.DB, jsonBytes []byte) (err error) {
	// we initialize our Users array
	var kleosTasks []PraticaKleos

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(jsonBytes, &kleosTasks)
	uid := (&uuid.UUID{}).String()

	return db.Transaction(func(tx *gorm.DB) error {
		taskCount := len(kleosTasks)
		for i, t := range kleosTasks {
			if len(t.DataDiArchiviazione) > 0 {
				// skip archived
				continue
			}
			task := models.Task{
				Code:          t.PosizioneArchivio,
				Name:          t.NomePratica,
				Description:   t.Descrizione,
				Archived:      len(t.DataDiArchiviazione) > 0,
				PaymentType:   new(models.PaymentType),
				PaymentAmount: new(float64),
			}
			*task.PaymentType = models.HourlyRate
			*task.PaymentAmount = 150

			fmt.Printf("[%v/%v] Importing %v ", i+1, taskCount, task.Name)

			if result := tx.Session(&gorm.Session{SkipHooks: true}).Set("userId", uid).Where(models.Task{Code: task.Code}).Assign(&task).FirstOrCreate(&task); result.Error != nil {
				fmt.Println("--> FAILED")
				return result.Error
			}
			fmt.Println("--> OK")

		}
		return nil
	})
}

func ImportTasksFromFile(db *gorm.DB, filename string) (err error) {

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

		return ImportTasks(db, byteValue)
	}

	return fmt.Errorf("cannot import %v", filename)
}

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("import tasks", cmd.Flag("file").Value)
		filename := cmd.Flag("file").Value.String()
		database.Connect()
		return ImportTasksFromFile(database.DB, filename)
	},
}

func init() {
	ImportCmd.AddCommand(tasksCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tasksCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tasksCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
