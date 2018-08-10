/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"encoding/base64"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	capicommon "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterclient "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	awsconfigv1 "github.com/enxebre/cluster-api-provider-aws/awsproviderconfig/v1alpha1"
	cov1 "github.com/enxebre/cluster-api-provider-aws/awsproviderconfig/v1alpha1"
	"github.com/openshift/cluster-operator/pkg/controller"
	clustoplog "github.com/openshift/cluster-operator/pkg/logging"
)

const (
	awsCredsSecretIDKey     = "awsAccessKeyId"
	awsCredsSecretAccessKey = "awsSecretAccessKey"
)

// Instance tag constants
// TODO: these do not match the case of the clustop or capi role names
const (
	hostTypeNode              = "node"
	hostTypeMaster            = "master"
	subHostTypeDefault        = "default"
	subHostTypeInfra          = "infra"
	subHostTypeCompute        = "compute"
	shutdownBehaviorTerminate = "terminate"
	shutdownBehaviorStop      = "stop"
)

var stateMask int64 = 0xFF

// Actuator is the AWS-specific actuator for the Cluster API machine controller
type Actuator struct {
	kubeClient    kubernetes.Interface
	clusterClient clusterclient.Interface
	//codecFactory            serializer.CodecFactory
	defaultAvailabilityZone string
	logger                  *log.Entry
	clientBuilder           func(kubeClient kubernetes.Interface, mSpec *cov1.MachineSetSpec, namespace, region string) (Client, error)
	//userDataGenerator       func(master, infra bool) (string, error)
	awsProviderConfigCodec *awsconfigv1.AWSProviderConfigCodec
	scheme                 *runtime.Scheme
	ignConfig              func(kubeClient kubernetes.Interface) (string, error)
}

func getWorkerRole() {

}

// NewActuator returns a new AWS Actuator
func NewActuator(kubeClient kubernetes.Interface, clusterClient clusterclient.Interface, logger *log.Entry, defaultAvailabilityZone string) *Actuator {

	scheme, err := awsconfigv1.NewScheme()
	if err != nil {
		return nil
	}
	codec, err := awsconfigv1.NewCodec()
	if err != nil {
		return nil
	}

	actuator := &Actuator{
		kubeClient:    kubeClient,
		clusterClient: clusterClient,
		//codecFactory:            coapi.Codecs,
		defaultAvailabilityZone: defaultAvailabilityZone,
		logger:                  logger,
		clientBuilder:           NewClient,
		//userDataGenerator:       generateUserData,
		awsProviderConfigCodec: codec,
		scheme:                 scheme,
		ignConfig:              getIgn,
	}
	return actuator
}

// Implements ProviderDeployer
func (a *Actuator) GetIP(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (string, error) {
	return "", nil
}

// Implements ProviderDeployer
func (a *Actuator) GetKubeConfig(cluster *clusterv1.Cluster, master *clusterv1.Machine) (string, error) {
	return "", nil
}

// Create runs a new EC2 instance
func (a *Actuator) Create(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	mLog := clustoplog.WithMachine(a.logger, machine)
	mLog.Info("creating machine")
	instance, err := a.CreateMachine(cluster, machine)
	if err != nil {
		mLog.Errorf("error creating machine: %v", err)
		return err
	}

	return a.updateStatus(machine, instance, mLog)
}

// CreateMachine starts a new AWS instance as described by the cluster and machine resources
func (a *Actuator) CreateMachine(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (*ec2.Instance, error) {
	mLog := clustoplog.WithMachine(a.logger, machine)
	// Extract cluster operator cluster
	awsClusterProviderConfig, err := a.awsProviderConfigCodec.ClusterProviderFromProviderConfig(cluster.Spec.ProviderConfig)
	if err != nil {
		return nil, err
	}

	if awsClusterProviderConfig.ClusterDeploymentSpec.Hardware.AWS == nil {
		return nil, fmt.Errorf("Cluster does not contain an AWS hardware spec")
	}

	awsProviderConfig, err := a.awsProviderConfigCodec.MachineProviderFromProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, err
	}

	region := awsClusterProviderConfig.ClusterDeploymentSpec.Hardware.AWS.Region
	mLog.Debugf("Obtaining EC2 client for region %q", region)
	client, err := a.clientBuilder(a.kubeClient, &awsProviderConfig.MachineSetSpec, machine.Namespace, region)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain EC2 client: %v", err)
	}

	if awsProviderConfig.VMImage.AWSImage == nil {
		return nil, fmt.Errorf("machine does not have an AWS image set")
	}

	// Get AMI to use
	amiName := *awsProviderConfig.VMImage.AWSImage

	mLog.Debugf("Describing AMI %s", amiName)
	imageIds := []*string{aws.String(amiName)}
	describeImagesRequest := ec2.DescribeImagesInput{
		ImageIds: imageIds,
	}
	describeAMIResult, err := client.DescribeImages(&describeImagesRequest)
	if err != nil {
		return nil, fmt.Errorf("error describing AMI %s: %v", amiName, err)
	}
	if len(describeAMIResult.Images) != 1 {
		return nil, fmt.Errorf("Unexpected number of images returned: %d", len(describeAMIResult.Images))
	}

	// Describe VPC
	vpcName := awsProviderConfig.ClusterID
	clusterName := strings.Split(vpcName, ".")[0]
	vpcNameFilter := "tag:Name"
	describeVpcsRequest := ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{{Name: &vpcNameFilter, Values: []*string{&vpcName}}},
	}
	describeVpcsResult, err := client.DescribeVpcs(&describeVpcsRequest)
	if err != nil {
		return nil, fmt.Errorf("Error describing VPC %s: %v", vpcName, err)
	}
	if len(describeVpcsResult.Vpcs) != 1 {
		return nil, fmt.Errorf("Unexpected number of VPCs: %d", len(describeVpcsResult.Vpcs))
	}
	vpcID := *(describeVpcsResult.Vpcs[0].VpcId)

	// Describe Subnet
	describeSubnetsRequest := ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(vpcID)}},
		},
	}
	// Filter by default availability zone if one was passed, otherwise, take the first subnet
	// that comes back.
	if len(a.defaultAvailabilityZone) > 0 {
		describeSubnetsRequest.Filters = append(describeSubnetsRequest.Filters, &ec2.Filter{Name: aws.String("availability-zone"), Values: []*string{aws.String(a.defaultAvailabilityZone)}})
	}
	describeSubnetsResult, err := client.DescribeSubnets(&describeSubnetsRequest)
	if err != nil {
		return nil, fmt.Errorf("Error describing Subnets for VPC %s: %v", vpcName, err)
	}
	if len(describeSubnetsResult.Subnets) == 0 {
		return nil, fmt.Errorf("Did not find a subnet")
	}

	// Determine security groups
	describeSecurityGroupsInput := buildDescribeSecurityGroupsInput(vpcID, vpcName, controller.MachineHasRole(machine, capicommon.MasterRole), awsProviderConfig.Infra)
	describeSecurityGroupsOutput, err := client.DescribeSecurityGroups(describeSecurityGroupsInput)
	if err != nil {
		return nil, err
	}

	var securityGroupIds []*string
	for _, g := range describeSecurityGroupsOutput.SecurityGroups {
		groupID := *g.GroupId
		securityGroupIds = append(securityGroupIds, &groupID)
	}

	// build list of networkInterfaces (just 1 for now)
	var networkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{
		{
			DeviceIndex:              aws.Int64(0),
			AssociatePublicIpAddress: aws.Bool(true),
			SubnetId:                 describeSubnetsResult.Subnets[0].SubnetId,
			Groups:                   securityGroupIds,
		},
	}

	// Set host type and sub-type to match what we do in pkg/ansible/generate.go. This is required
	// mainly for our Ansible code that dynamically generates the list of masters by searching for
	// AWS tags.
	hostType := hostTypeNode
	subHostType := subHostTypeCompute
	shutdownBehavior := shutdownBehaviorTerminate
	if controller.MachineHasRole(machine, capicommon.MasterRole) {
		hostType = hostTypeMaster
		subHostType = subHostTypeDefault
		shutdownBehavior = shutdownBehaviorStop
	}
	if awsProviderConfig.Infra {
		subHostType = subHostTypeInfra
	}
	mLog.WithFields(log.Fields{"hostType": hostType, "subHostType": subHostType}).Debugf("creating instance with host type")

	// Add tags to the created machine
	tagList := []*ec2.Tag{
		{Key: aws.String("clusterid"), Value: aws.String(vpcName)},
		{Key: aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", clusterName)), Value: aws.String("owned")},
		{Key: aws.String("tectonicClusterID"), Value: aws.String("447c6a4c-92a9-0266-3a23-9e3495006e24")},
		{Key: aws.String("Name"), Value: aws.String(machine.Name)},
	}
	tagInstance := &ec2.TagSpecification{
		ResourceType: aws.String("instance"),
		Tags:         tagList,
	}
	tagVolume := &ec2.TagSpecification{
		ResourceType: aws.String("volume"),
		Tags:         tagList[0:1],
	}

	ignConfig, err := a.ignConfig(a.kubeClient)
	if err != nil {
		return nil, fmt.Errorf("unable to obtain EC2 client: %v", err)
	}
	userDataEnc := base64.StdEncoding.EncodeToString([]byte(ignConfig))

	inputConfig := ec2.RunInstancesInput{
		ImageId:      describeAMIResult.Images[0].ImageId,
		InstanceType: aws.String(awsProviderConfig.Hardware.AWS.InstanceType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      aws.String(awsClusterProviderConfig.ClusterDeploymentSpec.Hardware.AWS.KeyPairName),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: aws.String(iamRole(clusterName)),
		},
		//BlockDeviceMappings: blkDeviceMappings,
		TagSpecifications: []*ec2.TagSpecification{tagInstance, tagVolume},
		NetworkInterfaces: networkInterfaces,
		UserData:          &userDataEnc,
		InstanceInitiatedShutdownBehavior: aws.String(shutdownBehavior),
	}

	runResult, err := client.RunInstances(&inputConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create EC2 instance: %v", err)
	}

	if runResult == nil || len(runResult.Instances) != 1 {
		mLog.Errorf("unexpected reservation creating instances: %v", runResult)
		return nil, fmt.Errorf("unexpected reservation creating instance")
	}

	//addInstanceToELB(runResult.Instances[0], "", client)
	return runResult.Instances[0], nil
}

// Delete deletes a machine and updates its finalizer
func (a *Actuator) Delete(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	mLog := clustoplog.WithMachine(a.logger, machine)
	mLog.Info("deleting machine")
	if err := a.DeleteMachine(machine); err != nil {
		mLog.Errorf("error deleting machine: %v", err)
		return err
	}
	return nil
}

// DeleteMachine deletes an AWS instance
func (a *Actuator) DeleteMachine(machine *clusterv1.Machine) error {
	mLog := clustoplog.WithMachine(a.logger, machine)

	awsProviderConfig, err := a.awsProviderConfigCodec.MachineProviderFromProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	if awsProviderConfig.ClusterHardware.AWS == nil {
		return fmt.Errorf("machine does not contain AWS hardware spec")
	}
	region := awsProviderConfig.ClusterHardware.AWS.Region
	client, err := a.clientBuilder(a.kubeClient, &awsProviderConfig.MachineSetSpec, machine.Namespace, region)
	if err != nil {
		return fmt.Errorf("error getting EC2 client: %v", err)
	}

	clusterId := awsProviderConfig.ClusterID
	instances, err := GetRunningInstances(machine, client, clusterId)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		mLog.Warnf("no instances found to delete for machine")
		return nil
	}

	return TerminateInstances(client, instances, mLog)
}

// Update attempts to sync machine state with an existing instance. Today this just updates status
// for details that may have changed. (IPs and hostnames) We do not currently support making any
// changes to actual machines in AWS. Instead these will be replaced via MachineDeployments.
func (a *Actuator) Update(cluster *clusterv1.Cluster, machine *clusterv1.Machine) error {
	mLog := clustoplog.WithMachine(a.logger, machine)
	mLog.Debugf("updating machine")

	awsClusterProviderConfig, err := a.awsProviderConfigCodec.ClusterProviderFromProviderConfig(cluster.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	if awsClusterProviderConfig.ClusterDeploymentSpec.Hardware.AWS == nil {
		return fmt.Errorf("Cluster does not contain an AWS hardware spec")
	}

	awsProviderConfig, err := a.awsProviderConfigCodec.MachineProviderFromProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return err
	}

	region := awsClusterProviderConfig.ClusterDeploymentSpec.Hardware.AWS.Region
	mLog.WithField("region", region).Debugf("obtaining EC2 client for region")
	client, err := a.clientBuilder(a.kubeClient, &awsProviderConfig.MachineSetSpec, machine.Namespace, region)
	if err != nil {
		return fmt.Errorf("unable to obtain EC2 client: %v", err)
	}

	instances, err := GetRunningInstances(machine, client, awsProviderConfig.ClusterID)
	mLog.Debugf("found %d instances for machine", len(instances))
	if err != nil {
		return err
	}

	// Parent controller should prevent this from ever happening by calling Exists and then Create,
	// but instance could be deleted between the two calls.
	if len(instances) == 0 {
		mLog.Warnf("attempted to update machine but no instances found")
		// Update status to clear out machine details.
		err := a.updateStatus(machine, nil, mLog)
		if err != nil {
			return err
		}
		return fmt.Errorf("attempted to update machine but no instances found")
	}
	newestInstance, terminateInstances := SortInstances(instances)

	// In very unusual circumstances, there could be more than one machine running matching this
	// machine name and cluster ID. In this scenario we will keep the newest, and delete all others.
	mLog = mLog.WithField("instanceID", *newestInstance.InstanceId)
	mLog.Debug("instance found")

	if len(instances) > 1 {
		err = TerminateInstances(client, terminateInstances, mLog)
		if err != nil {
			return err
		}

	}

	// We do not support making changes to pre-existing instances, just update status.
	return a.updateStatus(machine, newestInstance, mLog)
}

// Exists determines if the given machine currently exists. For AWS we query for instances in
// running state, with a matching name tag, to determine a match.
func (a *Actuator) Exists(cluster *clusterv1.Cluster, machine *clusterv1.Machine) (bool, error) {
	mLog := clustoplog.WithMachine(a.logger, machine)
	mLog.Debugf("checking if machine exists")

	awsProviderConfig, err := a.awsProviderConfigCodec.MachineProviderFromProviderConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return false, err
	}

	if awsProviderConfig.ClusterHardware.AWS == nil {
		return false, fmt.Errorf("machineSet does not contain AWS hardware spec")
	}
	region := awsProviderConfig.ClusterHardware.AWS.Region
	client, err := a.clientBuilder(a.kubeClient, &awsProviderConfig.MachineSetSpec, machine.Namespace, region)
	if err != nil {
		return false, fmt.Errorf("error getting EC2 client: %v", err)
	}

	instances, err := GetRunningInstances(machine, client, awsProviderConfig.ClusterID)
	if err != nil {
		return false, err
	}
	if len(instances) == 0 {
		mLog.Debug("instance does not exist")
		return false, nil
	}

	// If more than one result was returned, it will be handled in Update.
	mLog.Debug("instance exists")
	return true, nil
}

// updateStatus calculates the new machine status, checks if anything has changed, and updates if so.
func (a *Actuator) updateStatus(machine *clusterv1.Machine, instance *ec2.Instance, mLog log.FieldLogger) error {

	mLog.Debug("updating status")

	// Starting with a fresh status as we assume full control of it here.
	awsStatus, err := a.awsProviderConfigCodec.AWSMachineProviderStatusFromClusterAPIMachine(machine)
	if err != nil {
		return err
	}
	// Save this, we need to check if it changed later.
	origInstanceID := awsStatus.InstanceID

	// Instance may have existed but been deleted outside our control, clear it's status if so:
	if instance == nil {
		awsStatus.InstanceID = nil
		awsStatus.InstanceState = nil
		awsStatus.PublicIP = nil
		awsStatus.PublicDNS = nil
		awsStatus.PrivateIP = nil
		awsStatus.PrivateDNS = nil
	} else {
		awsStatus.InstanceID = instance.InstanceId
		awsStatus.InstanceState = instance.State.Name
		// Some of these pointers may still be nil (public IP and DNS):
		awsStatus.PublicIP = instance.PublicIpAddress
		awsStatus.PublicDNS = instance.PublicDnsName
		awsStatus.PrivateIP = instance.PrivateIpAddress
		awsStatus.PrivateDNS = instance.PrivateDnsName
	}
	mLog.Debug("finished calculating AWS status")

	if !controller.StringPtrsEqual(origInstanceID, awsStatus.InstanceID) {
		mLog.Debug("AWS instance ID changed, clearing LastELBSync to trigger adding to ELBs")
		awsStatus.LastELBSync = nil
	}

	awsStatusRaw, err := a.awsProviderConfigCodec.ClusterAPIMachineProviderStatusFromAWSMachineProviderStatus(awsStatus)
	if err != nil {
		mLog.Errorf("error encoding AWS provider status: %v", err)
		return err
	}

	machineCopy := machine.DeepCopy()
	machineCopy.Status.ProviderStatus = awsStatusRaw

	if !equality.Semantic.DeepEqual(machine.Status, machineCopy.Status) {
		mLog.Info("machine status has changed, updating")
		machineCopy.Status.LastUpdated = metav1.Now()

		_, err := a.clusterClient.ClusterV1alpha1().Machines(machineCopy.Namespace).UpdateStatus(machineCopy)
		if err != nil {
			mLog.Errorf("error updating machine status: %v", err)
			return err
		}
	} else {
		mLog.Debug("status unchanged")
	}
	return nil
}

func iamRole(clusterName string) string {
	return fmt.Sprintf("%s-master-profile", clusterName)
}

func buildDescribeSecurityGroupsInput(vpcID, vpcName string, isMaster, isInfra bool) *ec2.DescribeSecurityGroupsInput {
	groupNames := []*string{aws.String(vpcName)}
	groupNames = append(groupNames, aws.String(vpcName+"_worker_sg"))

	return &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{&vpcID}},
			{Name: aws.String("group-name"), Values: groupNames},
		},
	}
}
