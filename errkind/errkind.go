package errkind

// Kind implements the errors.Kind interface and
// identifies the category of an error.
type Kind uint8

const (
	Unspecified Kind = iota // Error that does not fall into any category.
	Invalid                 // Invalid operation on an item.
	Internal                // Internal error or inconsistency.
	IO                      // An OS level I/O error.
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
	}
	return "unknown error kind"
}
