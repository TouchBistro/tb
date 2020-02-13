package login

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"
)

type ECRLoginStrategy struct{}

func (s ECRLoginStrategy) Name() string {
	return "ECR"
}

func (s ECRLoginStrategy) Login() error {
	sess := session.New()
	stssvc := sts.New(sess)
	ecrsvc := ecr.New(sess)
	stsin := &sts.GetCallerIdentityInput{}
	result, err := stssvc.GetCallerIdentity(stsin)
	if err != nil {
		return errors.Wrap(err, "failed to get AWS account ID - try running aws configure.")
	}
	account := result.Account
	ecrin := &ecr.GetAuthorizationTokenInput{}
	authdata, err := ecrsvc.GetAuthorizationToken(ecrin)
	if err != nil {
		return errors.Wrap(err, "failed to get ECR login token - try running aws configure.")
	}
	token := *authdata.AuthorizationData[0].AuthorizationToken
	argString := fmt.Sprintf("login --username AWS --password-stdin https://%s.dkr.ecr.us-east-1.amazonaws.com", account)

	cmd := exec.Command("docker", argString)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "Couldn't open stdin to docker cli")
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "Could not start docker cli")
	}
	_, err = io.WriteString(stdin, token)
	return errors.Wrap(err, "docker login failed")
}
