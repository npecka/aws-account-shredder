package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	clientpkg "github.com/openshift/aws-account-shredder/pkg/aws"
	"github.com/openshift/aws-account-shredder/pkg/awsManager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeRest "k8s.io/client-go/rest"
)

const (
	accountID               = "" // it is 12 digit account id , the sub level account , for future use and debugging purpose
	sessionName             = ""
	awsCredsSecretIDKey     = "aws_access_key_id"
	awsCredsSecretAccessKey = "aws_secret_access_key"
	namespace               = "aws-account-shredder"             // change the namespace according to your environment. this is the namespace, from where secret has to retreived from
	secretName              = "aws-account-shredder-credentials" // the name of the secret to be read
)

var (
	supportedRegions = []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2", "ca-central-1", "eu-central-1", "eu-west-1", "eu-west-2", "eu-west-3", "ap-northeast-1", "ap-northeast-2", "ap-south-1", "ap-southeast-1", "ap-southeast-2", "sa-east-1"}
)

func main() {
	// creates the in-cluster config
	config, err := kubeRest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
	}

	// get secret "aws-account-shredder-credentials" from namespace "aws-account-shredder"
	secrets, err := clientset.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("ERROR:", err)
	}

	accessKeyID, ok := secrets.Data[awsCredsSecretIDKey]
	if !ok {
		fmt.Println("ERROR:", err)
	}
	secretAccessKey, ok := secrets.Data[awsCredsSecretAccessKey]
	if !ok {
		fmt.Println("ERROR:", err)
	}

	// creating a new AWSclient with the information extracted from the secret file
	client, err := clientpkg.NewClient(string(accessKeyID), string(secretAccessKey), "", "us-east-1")
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	for {

		// assuming roles for the given AccountID
		RoleArnParameter := "arn:aws:iam::" + accountID + ":role/OrganizationAccountAccessRole"
		assumedRole, err := client.AssumeRole(&sts.AssumeRoleInput{RoleArn: aws.String(RoleArnParameter), RoleSessionName: aws.String(sessionName)})
		if err != nil {
			fmt.Println("ERROR:", err)

		}
		assumedAccessKey := *assumedRole.Credentials.AccessKeyId
		assumedSecretKey := *assumedRole.Credentials.SecretAccessKey
		assumedSessionToken := *assumedRole.Credentials.SessionToken

		for _, region := range supportedRegions {
			fmt.Println("\n EC2 instances in region ", region)
			assumedRoleClient, err := clientpkg.NewClient(assumedAccessKey, assumedSecretKey, assumedSessionToken, region)
			if err != nil {
				fmt.Println("ERROR:", err)
			}

			awsManager.CleanS3Instances(assumedRoleClient)
			awsManager.CleanEc2Instances(assumedRoleClient)
		}

	}
}