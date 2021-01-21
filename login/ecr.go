package login

import (
	"context"
	"encoding/base64"
	"io"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/pkg/errors"
)

type ECRLoginStrategy struct{}

func (s ECRLoginStrategy) Name() string {
	return "ECR"
}

func (ECRLoginStrategy) Login() error {
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start aws session - try running aws configure.")
	}

	client := ecr.NewFromConfig(conf)
	output, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return errors.Wrap(err, "failed to get ECR login token - try running aws configure.")
	}

	authData := output.AuthorizationData[0]
	tokenData, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return errors.Wrap(err, "failed to decode ECR login token")
	}

	// Token is in the from username:password, need to grab just the password
	token := strings.Split(string(tokenData), ":")[1]

	cmd := exec.Command("docker", "login", "--username", "AWS", "--password-stdin", *authData.ProxyEndpoint)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "Couldn't open stdin to docker cli")
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "Could not start docker cli")
	}

	_, err = io.WriteString(stdin, token)
	if err != nil {
		return errors.Wrap(err, "failed to write ecr login password to stdin")
	}

	err = stdin.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close stdin")
	}

	err = cmd.Wait()
	return errors.Wrap(err, "failed to run docker login")
}
