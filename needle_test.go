package main

import (
	"fmt"
	"testing"
)

func Test_Needle(t *testing.T) {

	needles := []string{
		"path:pub/static/frontend/BodyAndBeach/store/fr_CA/Altima_Lookbookslider/js/jquery.cycle2_1.js",
		"path:app/code/Magedelight/GeoIp/Controller/Adminhtml/Currencymapping/Delete.php",
	}
	dbpath := "m2.db"

	db := loadDB(dbpath)
	fmt.Println("Loaded entries:", len(db))

	for k, _ := range db {
		fmt.Println("first entry", k)
		break
	}

	for _, needle := range needles {
		checksum := hash([]byte(needle))
		hash := fmt.Sprintf("%x", checksum)
		_, ok := db[checksum]
		fmt.Println(ok, hash)
	}
}
