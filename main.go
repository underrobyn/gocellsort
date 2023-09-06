package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type CSVRow struct {
	ID            uint `gorm:"primaryKey"`
	Radio         string
	MCC           uint16
	MNC           uint16
	TAC           uint16
	PCI           uint16
	Lon           float64
	Lat           float64
	Range         uint32
	Samples       uint32
	Changeable    bool
	Created       uint32
	Updated       uint32
	AverageSignal int16
	ENodeB        uint32
	SectorID      uint16
}

func extractSectorAndENB(cid uint64) (sector uint16, enb uint32) {
	sector = uint16(cid % 256)
	enb = uint32(cid / 256)
	return sector, enb
}

func parseCSVRow(record []string) (CSVRow, error) {
	var row CSVRow

	row.Radio = record[0]

	var err error
	var mccValue uint64
	mccValue, err = strconv.ParseUint(record[1], 10, 64)
	row.MCC = uint16(mccValue)
	if err != nil {
		return row, err
	}

	var mncValue uint64
	mncValue, err = strconv.ParseUint(record[2], 10, 64)
	row.MNC = uint16(mncValue)
	if err != nil {
		return row, err
	}

	var tacValue uint64
	tacValue, err = strconv.ParseUint(record[3], 10, 64)
	row.TAC = uint16(tacValue)
	if err != nil {
		fmt.Println("No TAC")
		return row, err
	}

	var pciValue uint64 = 0
	if record[5] != "" {
		pciValue, err = strconv.ParseUint(record[5], 10, 64)
		if err != nil {
			return row, err
		}
	}
	row.PCI = uint16(pciValue)

	row.Lon, err = strconv.ParseFloat(record[6], 64)
	if err != nil {
		return row, err
	}

	row.Lat, err = strconv.ParseFloat(record[7], 64)
	if err != nil {
		return row, err
	}

	var rangeValue uint64
	rangeValue, err = strconv.ParseUint(record[8], 10, 32)
	row.Range = uint32(rangeValue)
	if err != nil {
		return row, err
	}

	var samplesValue uint64
	samplesValue, err = strconv.ParseUint(record[9], 10, 32)
	row.Samples = uint32(samplesValue)
	if err != nil {
		fmt.Println("No samples")
		return row, err
	}

	row.Changeable, err = strconv.ParseBool(record[10])
	if err != nil {
		return row, err
	}

	var createdValue uint64
	createdValue, err = strconv.ParseUint(record[11], 10, 32)
	row.Created = uint32(createdValue)
	if err != nil {
		return row, err
	}

	var updatedValue uint64
	updatedValue, err = strconv.ParseUint(record[12], 10, 32)
	row.Updated = uint32(updatedValue)
	if err != nil {
		return row, err
	}

	var averageSignalValue int64 = 0
	if record[13] != "" {
		averageSignalValue, err = strconv.ParseInt(record[13], 10, 16)
		if err != nil {
			return row, err
		}
	}
	row.AverageSignal = int16(averageSignalValue)

	cellID, err := strconv.ParseUint(record[4], 10, 32)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Decompose the cell ID
	sectorID, enodeb := extractSectorAndENB(cellID)

	if mccValue == 234 {
		fmt.Printf("MCC: %d, MNC: %d, eNB: %d, sector: %d\n", mccValue, mncValue, enodeb, sectorID)
	}

	row.ENodeB = enodeb
	row.SectorID = sectorID

	return row, nil
}

func readAndParseCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

func main() {
	err := godotenv.Load()

	filePath := "MLS-full-cell-export-2023-08-16T000000.csv"

	// Get database credentials from environment variables
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	// Construct the DSN
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=%s", dbUser, dbPassword, dbName, dbHost, dbSSLMode)

	// Create a database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// AutoMigrate will create the table if it doesn't exist
	err = db.AutoMigrate(&CSVRow{})
	if err != nil {
		return
	}

	// Read and parse the CSV file
	data, err := readAndParseCSV(filePath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Create a list to store the CSV rows
	var csvRows []CSVRow

	for _, record := range data {
		if record[0] != "LTE" {
			continue
		}

		row, err := parseCSVRow(record)
		if err != nil {
			fmt.Println("Error parsing row:", err)
			continue
		}

		csvRows = append(csvRows, row)

		result := db.Create(&row)
		if result.Error != nil {
			log.Fatal(result.Error)
		}
	}

	//for _, row := range csvRows {
	//	fmt.Printf("Radio: %s, MCC: %d, MNC: %d, TAC: %d, PCI: %d, Lon: %f, Lat: %f, Range: %d, Samples: %d, Changeable: %t, Created: %d, Updated: %d, AverageSignal: %d, ENodeB: %d, SectorID: %d\n",
	//		row.Radio, row.MCC, row.MNC, row.TAC, row.PCI, row.Lon, row.Lat, row.Range, row.Samples, row.Changeable, row.Created, row.Updated, row.AverageSignal, row.ENodeB, row.SectorID)
	//}
}
