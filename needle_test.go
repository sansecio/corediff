package main

import (
	"fmt"
	"hash/crc32"
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

	for _, needle := range needles {
		checksum := crc32.ChecksumIEEE([]byte(needle))
		hash := fmt.Sprintf("%x", checksum)
		_, ok := db[checksum]
		fmt.Println(ok, hash)
	}
}
