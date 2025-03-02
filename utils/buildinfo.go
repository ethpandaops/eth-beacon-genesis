package utils

import "fmt"

var BuildVersion string
var BuildRelease string
var Buildtime string

func GetBuildVersion() string {
	if BuildRelease == "" {
		return fmt.Sprintf("git-%v", BuildVersion)
	}

	return fmt.Sprintf("%v (git-%v)", BuildRelease, BuildVersion)
}
