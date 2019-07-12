package release

import (
	"fmt"
)

var (
	version = "dev"
	commit  = "none"
)

func VersionString() string {
	return fmt.Sprintf("%s-%s", version, commit)
}
