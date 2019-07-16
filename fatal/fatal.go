// TODO: Make stack trace user-configurable in tbrc
package fatal

import (
	"fmt"
	"os"
)

func ExitErr(err error, message string) {
	fmt.Fprintf(os.Stderr, message+"\n")

	fmt.Fprintf(os.Stderr, "Error: %+v\n", err)

	os.Exit(1)
}

func ExitErrf(err error, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Println()

	fmt.Fprintf(os.Stderr, "Error: %+v\n", err)

	os.Exit(1)
}

func Exit(message string) {
	fmt.Fprintf(os.Stderr, message+"\n")
	os.Exit(1)
}

func Exitf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)

	os.Exit(1)
}
