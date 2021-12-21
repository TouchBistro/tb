package login

import (
	"context"
	"encoding/base64"
	"io"
	"os/exec"
	"strings"

	"github.com/TouchBistro/goutils/errors"
	"github.com/TouchBistro/tb/errkind"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

type ecrStrategy struct{}

func (ecrStrategy) Name() string {
	return "ECR"
}

func (ecrStrategy) Login(ctx context.Context) error {
	const op = errors.Op("login.ecrStrategy.Login")
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.AWS,
			Reason: "failed to load AWS configuration",
			Op:     op,
		})
	}
	ecrClient := ecr.NewFromConfig(cfg)
	output, err := ecrClient.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.AWS,
			Reason: "unable to get ECR authorization token",
			Op:     op,
		})
	}

	authData := output.AuthorizationData[0]
	tokenData, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.AWS,
			Reason: "failed to decode ECR authorization token",
			Op:     op,
		})
	}

	// Token is in the from username:password, need to grab just the password
	token := strings.Split(string(tokenData), ":")[1]
	cmd := exec.CommandContext(ctx, "docker", "login", "--username", "AWS", "--password-stdin", *authData.ProxyEndpoint)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "couldn't open stdin to docker cli",
			Op:     op,
		})
	}
	// Make sure we close even if we return with an error
	defer stdin.Close()
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "couldn't start docker cli",
			Op:     op,
		})
	}
	if _, err := io.WriteString(stdin, token); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: "failed to write ECR login password to stdin",
			Op:     op,
		})
	}
	// Manually close to have docker continue
	if err := stdin.Close(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.IO,
			Reason: "failed to close stdin",
			Op:     op,
		})

	}
	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, errors.Meta{
			Kind:   errkind.Docker,
			Reason: "failed to run docker login",
			Op:     op,
		})
	}
	return nil
}
