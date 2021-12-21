package login

import (
	"context"
	"fmt"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
)

// Strategy defines provides functionality for logging into a service.
type Strategy interface {
	Name() string
	Login(ctx context.Context) error
}

func ParseStrategies(strategyNames []string) ([]Strategy, error) {
	strategies := make([]Strategy, len(strategyNames))
	for i, s := range strategyNames {
		switch strings.ToLower(s) {
		case "ecr":
			strategies[i] = ecrStrategy{}
		case "npm":
			strategies[i] = npmStrategy{}
		default:
			return nil, errors.New(
				errkind.Invalid,
				fmt.Sprintf("unknown login strategy %s", s),
				"login.ParseStrategies",
			)
		}
	}
	return strategies, nil
}
