package common

import (
	"os"
)

var (

	// Project is being unit tested
	IsTest bool

	// Currently running inside CI
	IsCI = os.Getenv("CI") == "true"
)
