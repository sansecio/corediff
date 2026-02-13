package main

type dbCmd struct {
	Add   dbAddArg   `command:"add" description:"Add files or dirs to the database"`
	Merge dbMergeArg `command:"merge" description:"Merge databases"`
	Info  dbInfoArg  `command:"info" description:"Show database information"`
}

var dbCommand dbCmd

func init() {
	cli.AddCommand("db", "Database operations", "Add, merge, and manage hash databases", &dbCommand)
}
