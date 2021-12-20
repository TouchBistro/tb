package errkind

// Kind implements the errors.Kind interface and
// identifies the category of an error.
type Kind uint8

const (
	Unspecified   Kind = iota // Error that does not fall into any category.
	Invalid                   // Invalid operation on an item.
	Internal                  // Internal error or inconsistency.
	IO                        // An OS level I/O error.
	Git                       // A Git operation returned an error.
	Docker                    // A Docker operation returned an error.
	DockerCompose             // A docker-compose operation returned an error.
	Simulator                 // A iOS simulator operation returned an error.
	AWS                       // An AWS operation returned an error.
)

func (k Kind) Kind() string {
	switch k {
	case Unspecified:
		return "unspecified error"
	case Invalid:
		return "invalid operation"
	case Internal:
		return "internal error"
	case IO:
		return "I/O error"
	case Git:
		return "git error"
	case Docker:
		return "docker error"
	case DockerCompose:
		return "docker-compose error"
	case Simulator:
		return "iOS simulator error"
	case AWS:
		return "AWS error"
	}
	return "unknown error kind"
}
