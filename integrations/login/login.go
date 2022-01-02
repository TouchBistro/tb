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

func ParseStrategy(strategyName string) (Strategy, error) {
	switch strings.ToLower(strategyName) {
	case "ecr":
		return ecrStrategy{}, nil
	case "npm":
		return npmStrategy{}, nil
	}
	return nil, errors.New(
		errkind.Invalid,
		fmt.Sprintf("unknown login strategy %s", strategyName),
		"login.ParseStrategy",
	)
}
