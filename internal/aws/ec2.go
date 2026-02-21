package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Resources tracks everything we create so teardown knows what to clean up.
type Resources struct {
	InstanceID      string
	SecurityGroupID string
	PublicIP        string
}

func getDefaultVPC(ctx context.Context, client *ec2.Client) (string, error) {
	out, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{Name: aws.String("isDefault"), Values: []string{"true"}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("describing VPCs: %w", err)
	}
	if len(out.Vpcs) == 0 {
		return "", fmt.Errorf("no default VPC found — Mayfly requires a default VPC")
	}
	return aws.ToString(out.Vpcs[0].VpcId), nil
}

func createSecurityGroup(ctx context.Context, client *ec2.Client, vpcID string) (string, error) {
	name := fmt.Sprintf("mayfly-%d", time.Now().UnixMilli())

	sg, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		Description: aws.String("Mayfly ephemeral exit node - safe to delete"),
		VpcId:       aws.String(vpcID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(name)},
					{Key: aws.String("mayfly"), Value: aws.String("true")},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("creating security group: %w", err)
	}

	sgID := aws.ToString(sg.GroupId)

	_, err = client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("udp"),
				FromPort:   aws.Int32(41641),
				ToPort:     aws.Int32(41641),
				IpRanges:   []types.IpRange{{CidrIp: aws.String("0.0.0.0/0"), Description: aws.String("Tailscale WireGuard")}},
			},
		},
	})
	if err != nil {
		return sgID, fmt.Errorf("authorizing ingress: %w", err)
	}

	return sgID, nil
}

// Provision creates a security group and launches an EC2 instance.
// It returns a Resources struct for teardown. If provisioning fails partway,
// the caller should still call Teardown with whatever Resources were populated.
func Provision(ctx context.Context, cfg aws.Config, amiID, instanceType, userData string) (*Resources, error) {
	client := ec2.NewFromConfig(cfg)
	res := &Resources{}

	vpcID, err := getDefaultVPC(ctx, client)
	if err != nil {
		return res, err
	}

	sgID, err := createSecurityGroup(ctx, client, vpcID)
	res.SecurityGroupID = sgID
	if err != nil {
		return res, err
	}

	runOut, err := client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:      aws.String(amiID),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		SecurityGroupIds: []string{sgID},
		UserData:     aws.String(userData),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String("mayfly-exit")},
					{Key: aws.String("mayfly"), Value: aws.String("true")},
				},
			},
		},
	})
	if err != nil {
		return res, fmt.Errorf("launching instance: %w", err)
	}

	res.InstanceID = aws.ToString(runOut.Instances[0].InstanceId)

	// Wait for instance to reach running state.
	waiter := ec2.NewInstanceRunningWaiter(client)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{res.InstanceID},
	}, 5*time.Minute)
	if err != nil {
		return res, fmt.Errorf("waiting for instance to start: %w", err)
	}

	// Fetch the public IP now that the instance is running.
	desc, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{res.InstanceID},
	})
	if err == nil && len(desc.Reservations) > 0 && len(desc.Reservations[0].Instances) > 0 {
		res.PublicIP = aws.ToString(desc.Reservations[0].Instances[0].PublicIpAddress)
	}

	return res, nil
}

// Teardown terminates the instance and deletes the security group.
// It uses context.Background() internally so cleanup always completes.
func Teardown(cfg aws.Config, res *Resources) error {
	ctx := context.Background()
	client := ec2.NewFromConfig(cfg)
	var firstErr error

	if res.InstanceID != "" {
		_, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{res.InstanceID},
		})
		if err != nil {
			firstErr = fmt.Errorf("terminating instance: %w", err)
		} else {
			// Wait for termination before deleting the SG (SG can't be deleted while in use).
			waiter := ec2.NewInstanceTerminatedWaiter(client)
			if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{res.InstanceID},
			}, 5*time.Minute); err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("waiting for instance termination: %w", err)
				}
			}
		}
	}

	if res.SecurityGroupID != "" {
		_, err := client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(res.SecurityGroupID),
		})
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("deleting security group: %w", err)
		}
	}

	return firstErr
}
