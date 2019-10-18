package color

const (
	red     = "\x1b[31m"
	green   = "\x1b[32m"
	yellow  = "\x1b[33m"
	blue    = "\x1b[34m"
	magenta = "\x1b[35m"
	cyan    = "\x1b[36m"
	reset   = "\x1b[0m"
)

// Red creates a red colored string
func Red(str string) string {
	return red + str + reset
}

// Green creates a green colored string
func Green(str string) string {
	return green + str + reset
}

// Yellow creates a yellow colored string
func Yellow(str string) string {
	return yellow + str + reset
}

// Blue creates a blue colored string
func Blue(str string) string {
	return blue + str + reset
}

// Magenta creates a magenta colored string
func Magenta(str string) string {
	return magenta + str + reset
}

// Cyan creates a cyan colored string
func Cyan(str string) string {
	return cyan + str + reset
}
