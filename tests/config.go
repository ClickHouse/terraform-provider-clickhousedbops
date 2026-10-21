package tests

import _ "embed"

//go:embed config-single.xml
var ConfigSingle string

//go:embed config-single-read-write-grants.xml
var ConfigSingleReadWriteGrants string
