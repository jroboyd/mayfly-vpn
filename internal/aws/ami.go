package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const al2023AMIParameter = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"

func LookupAMI(ctx context.Context, cfg aws.Config) (string, error) {
	client := ssm.NewFromConfig(cfg)

	out, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(al2023AMIParameter),
	})
	if err != nil {
		return "", fmt.Errorf("looking up AMI: %w", err)
	}

	return aws.ToString(out.Parameter.Value), nil
}
