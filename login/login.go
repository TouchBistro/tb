package login

import (
	"github.com/pkg/errors"
)

type LoginStrategy interface {
	Name() string
	Login() error
}

func ParseStrategies(strategyNames []string) ([]LoginStrategy, error) {
	strategies := make([]LoginStrategy, len(strategyNames))
	for i, s := range strategyNames {
		switch s {
		case "ecr":
			strategies[i] = ECRLoginStrategy{}
		case "npm":
			strategies[i] = NPMLoginStrategy{}
		default:
			return nil, errors.Errorf("Invalid login strategy %s", s)
		}
	}

	return strategies, nil
}
