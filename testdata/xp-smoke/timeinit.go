package main

import "time"

// Package-level Zone and Format match BurntSushi/toml init (issue 10).
// time.Now().Unix does not call initLocal, so those calls must stay here.
var (
	issue10Now           = time.Now()
	issue10Name          string
	issue10Offset        int
	issue10RFC3339       string
	issue10UnixLocal     int64
	issue10LocationName  string
)

func init() {
	issue10Name, issue10Offset = issue10Now.Zone()
	issue10RFC3339 = issue10Now.Format(time.RFC3339)
	issue10UnixLocal = issue10Now.Unix()
	issue10LocationName = issue10Now.Location().String()
}
