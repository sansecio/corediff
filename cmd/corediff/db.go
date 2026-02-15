package main

type dbCmd struct {
	Database string     `short:"d" long:"database" description:"Hash database path" required:"true"`
	CacheDir string     `short:"c" long:"cache-dir" default:"./cache" description:"Cache directory for git clones and zip downloads"`
	Index    dbIndexArg `command:"index" description:"Index files or dirs into the database"`
	Merge    dbMergeArg `command:"merge" description:"Merge databases"`
	Info     dbInfoArg  `command:"info" description:"Show database information"`
}

var dbCommand dbCmd

func init() {
	cli.AddCommand("db", "Database operations", "Index, merge, and manage hash databases", &dbCommand)
}
