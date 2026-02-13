package main

type dbCmd struct {
	Database string     `short:"d" long:"database" description:"Hash database path" required:"true"`
	CacheDir string     `short:"c" long:"cache-dir" description:"Cache directory for git clones and zip downloads (default: temp dir)"`
	Add      dbAddArg   `command:"add" description:"Add files or dirs to the database"`
	Merge    dbMergeArg `command:"merge" description:"Merge databases"`
	Info     dbInfoArg  `command:"info" description:"Show database information"`
}

var dbCommand dbCmd

func init() {
	cli.AddCommand("db", "Database operations", "Add, merge, and manage hash databases", &dbCommand)
}
