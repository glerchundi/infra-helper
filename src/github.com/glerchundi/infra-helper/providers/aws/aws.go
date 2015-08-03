// Copyright (c) 2015 Gorka Lerchundi Osa. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/glerchundi/infra-helper/util"
)

type AwsMember struct {
	name string
	ipAddress string
}

func (awsMember *AwsMember) GetName() string {
	return awsMember.name
}

func (awsMember *AwsMember) GetIPAddress() string {
	return awsMember.ipAddress
}

func New() *Aws {
	return &Aws{}
}

type Aws struct {
}

func (aws *Aws) GetInstanceId() (string, error) {
	// Instance Id
	instanceId, err := util.HttpGet("http://169.254.169.254/latest/meta-data/instance-id")
	if err != nil {
		return "", err
	}

	return instanceId, nil
}

func (aws *Aws) GetInstancePrivateAddress() (string, error) {
	// Local IPv4 (Private Address)
	localIp, err := util.HttpGet("http://169.254.169.254/latest/meta-data/local-ipv4")
	if err != nil {
		return "", err
	}

	return localIp, nil
}

func (aws *Aws) GetClusterMembers() (map[string]string, error) {
	return GetClusterMembersByFunc(func(region string)(*autoscaling.Group, error) {
		// Instance Id
		instanceId, err := aws.GetInstanceId()
		if err != nil {
			return nil, err
		}
		return findAutoscalingGroupInstanceIdBelongs(instanceId, region)
	})
}

func (aws *Aws) GetClusterMembersByName(name string) (map[string]string, error) {
	return GetClusterMembersByFunc(func(region string)(*autoscaling.Group, error) {
		return findAutoscalingGroupByName(name, region)
	})
}

func GetClusterMembersByFunc(findAutoscalingGroup func(string)(*autoscaling.Group, error)) (map[string]string, error) {
	// Availability Zone
	availabilityZone, err := util.HttpGet("http://169.254.169.254/latest/meta-data/placement/availability-zone")
	if err != nil {
		return nil, err
	}

	// Region
	region := availabilityZone[:len(availabilityZone)-1]

	// Find which is the autoscaling group
	autoscalingGroup, err := findAutoscalingGroup(region)
	if err != nil {
		return nil, err
	}

	// Create list of instance identifiers
	instanceIds := make([]*string, 0)
	for _, i := range autoscalingGroup.Instances {
		instanceIds = append(instanceIds, i.InstanceID)
	}

	// Find EC2 instance properties
	privateAddresses, err := findEC2InstancesPrivateAddresses(instanceIds, region)
	if err != nil {
		return nil, err
	}

	return privateAddresses, nil
}

func findAutoscalingGroupInstanceIdBelongs(instanceId, region string) (*autoscaling.Group, error) {
	return findAutoscalingGroupByFunc(region, func(asg *autoscaling.Group)bool {
		for _, instance := range asg.Instances {
			if *(instance.InstanceID) == instanceId {
				return true
			}
		}
		return false
	})
}

func findAutoscalingGroupByName(name, region string) (*autoscaling.Group, error) {
	return findAutoscalingGroupByFunc(region, func(asg *autoscaling.Group)bool {
		return *asg.AutoScalingGroupName == name
	})
}

func findAutoscalingGroupByFunc(region string, predicate func(*autoscaling.Group)bool) (*autoscaling.Group, error) {
	var autoscalingGroup *autoscaling.Group

	svc := autoscaling.New(&aws.Config{Region: region})
	out, err := svc.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, err
	}

	for _, asg := range out.AutoScalingGroups {
		if predicate(asg) {
			autoscalingGroup = asg
			break
		}
	}

	if autoscalingGroup == nil {
		return nil, errors.New("failed to get the auto scaling group name")
	}

	return autoscalingGroup, nil
}

func findEC2InstancesPrivateAddresses(instanceIds []*string, region string) (map[string]string, error) {
	svc := ec2.New(&aws.Config{Region: region})
	out, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIDs: instanceIds})
	if err != nil {
		return nil, err
	}

	privateAddresses := make(map[string]string)
	for _, reservation := range out.Reservations {
		for _, instance := range reservation.Instances {
			privateAddresses[*instance.InstanceID] = *instance.PrivateIPAddress
		}
	}

	return privateAddresses, nil
}
