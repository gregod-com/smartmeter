package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var sqliteDB *gorm.DB

// Initialize the BoltDB
func initDB() {
	var err error
	sqliteDB, err = gorm.Open(sqlite.Open(config.DatabasePath), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	sqliteDB.AutoMigrate(&GasMeterReading{})
	calculateAndSaveDeltas(sqliteDB)
}

func createGasMeterReading(object *GasMeterReading) error {
	tx := sqliteDB.Create(object)
	calculateAndSaveDeltas(tx)
	return tx.Error
}

func updateGasMeterReading(object *GasMeterReading) error {
	tx := sqliteDB.Save(object)
	calculateAndSaveDeltas(tx)
	return tx.Error
}

func deleteGasMeterReading(object *GasMeterReading, id string) error {
	tx := sqliteDB.Delete(object, "id = ?", id)
	calculateAndSaveDeltas(sqliteDB)
	return tx.Error
}

func calculateAndSaveDeltas(tx *gorm.DB) error {
	var allReadings []GasMeterReading
	tx.Order("date" + " " + "ASC").Find(&allReadings)

	for i := range allReadings {
		if i == 0 {
			allReadings[i].DeltaMeasurement = 0
			continue
		}
		allReadings[i].DeltaMeasurement = allReadings[i].Measurement - allReadings[i-1].Measurement

		allReadings[i].DeltaDays = allReadings[i].Date.Sub(allReadings[i-1].Date).Hours() / 24
		allReadings[i].AverageSinceLast = allReadings[i].DeltaMeasurement / allReadings[i].DeltaDays
		// search older entries until the first that is older that 1 day
		if allReadings[i].DeltaDays > 1 {
			allReadings[i].DailyAverage = allReadings[i].DeltaMeasurement / allReadings[i].DeltaDays
			continue
		}
		// iterate over older entries until the first that is older that 1 day
		for j := i - 1; j >= 0; j-- {
			if allReadings[i].Date.Sub(allReadings[j].Date).Hours() > 24 {
				allReadings[i].DailyAverage = (allReadings[i].Measurement - allReadings[j].Measurement) / (allReadings[i].Date.Sub(allReadings[j].Date).Hours() / 24)
				break
			}
		}
	}
	// Save records without triggering hooks
	if err := tx.Session(&gorm.Session{SkipHooks: true}).Save(&allReadings).Error; err != nil {
		return err
	}
	return nil
}

func read(object interface{}, id string) error {
	return sqliteDB.First(object, "id = ?", id).Error
}

func getAll(object interface{}) error {
	return sqliteDB.Find(object).Error
}

func getPaginated(object interface{}, start, end int, sort, order string) error {
	return sqliteDB.Limit(end - start).Offset(start).Order(sort + " " + order).Find(object).Error
}

func count(object interface{}) (int64, error) {
	var count int64
	err := sqliteDB.Model(object).Count(&count).Error
	return count, err
}
