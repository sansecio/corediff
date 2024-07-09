package main

import (
	"fmt"
	"os"
)

type addArg struct {
	globalOpt
	Database string `short:"d" long:"database" description:"Hash database path (default: download Sansec database)" required:"true"`
}

var addArgs addArg

func init() {
	_, _ = cli.AddCommand("add", "Add files, dirs or PHP packages to the database", "", &addArgs)
}

func (a *addArg) Execute(args []string) error {
	fmt.Println("Do I have a db?", a.Database)

	db, err := loadDB(a.Database)
	if os.IsNotExist(err) {
		db = newDB()
	}
	if err != nil {
		return err
	}

	for _, p := range args {
		err := addPath(p, db)
		if err != nil {
			return err
		}
	}

	return nil
}
