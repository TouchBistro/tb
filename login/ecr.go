package login

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/pkg/errors"
)

type ECRLoginStrategy struct{}

func (s ECRLoginStrategy) Name() string {
	return "ECR"
}

func (s ECRLoginStrategy) Login() error {
	opts := session.Options{SharedConfigState: session.SharedConfigEnable}
	sess, err := session.NewSessionWithOptions(opts)
	if err != nil {
		return errors.Wrap(err, "failed to start aws session - try running aws configure.")
	}
	ecrsvc := ecr.New(sess)

	result, err := ecrsvc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return errors.Wrap(err, "failed to get ECR login token - try running aws configure.")
	}

	authData := *result.AuthorizationData[0]
	token := authData.AuthorizationToken
	endpoint := authData.ProxyEndpoint
	argString := fmt.Sprintf("login --username AWS --password-stdin %s", endpoint)

	cmd := exec.Command("docker", argString)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "Couldn't open stdin to docker cli")
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "Could not start docker cli")
	}
	_, err = io.WriteString(stdin, *token)
	cmd.Wait()
	return errors.Wrap(err, "docker login failed")
}
