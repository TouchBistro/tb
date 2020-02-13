package login

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/pkg/errors"
)

type ECRLoginStrategy struct{}

func (s ECRLoginStrategy) Name() string {
	return "ECR"
}

func (s ECRLoginStrategy) Login() error {
	buf := &bytes.Buffer{}
	err := command.Exec("aws", strings.Fields("ecr get-login --region us-east-1 --no-include-email"), "aws-ecr-login", func(cmd *exec.Cmd) {
		cmd.Stdout = buf
	})
	if err != nil {
		return errors.Wrap(err, "executing aws ecr get-login failed - try running aws configure.")
	}

	dockerLoginArgs := strings.Fields(buf.String())
	err = command.Exec(dockerLoginArgs[0], dockerLoginArgs[1:], "ecr-login")
	return errors.Wrap(err, "docker login failed")
}
