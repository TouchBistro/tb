package login

import (
	"os/exec"
	"strings"

	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

type ECRLoginStrategy struct{}

func (s ECRLoginStrategy) Name() string {
	return "ECR"
}

func (s ECRLoginStrategy) Login() error {
	out, err := exec.Command("aws", strings.Fields("ecr get-login --region us-east-1 --no-include-email")...).Output()
	if err != nil {
		return errors.Wrap(err, "executing aws ecr get-login failed - try running aws configure.")
	}

	dockerLoginArgs := strings.Fields(string(out))
	err = util.Exec("ecr-login", dockerLoginArgs[0], dockerLoginArgs[1:]...)
	return errors.Wrap(err, "docker login failed")
}
