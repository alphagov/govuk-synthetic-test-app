package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// RoleWrapper encapsulates AWS Identity and Access Management (IAM) role actions
// used in the examples.
// It contains an IAM service client that is used to perform role actions.
type RoleWrapper struct {
	IamClient *iam.Client
}

// GetRole gets data about a role.
func (wrapper RoleWrapper) GetRole(ctx context.Context, roleName string) (*types.Role, error) {
	var role *types.Role
	result, err := wrapper.IamClient.GetRole(ctx,
		&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		log.Printf("Couldn't get role %v. Here's why: %v\n", roleName, err)
	} else {
		role = result.Role
	}
	return role, err
}

func AssumedRole(ctx context.Context) string {
	noPermsConfig, err := config.LoadDefaultConfig(ctx)

	iamClient := iam.NewFromConfig(noPermsConfig)
	roleWrapper := RoleWrapper{IamClient: iamClient}
	role, err := roleWrapper.GetRole(ctx, "synthetic-test-assumed")
	if err != nil {
		log.Printf("Couldn't get role %v. Here's why: %v\n", "synthetic-test-assumed", err)
		return err.Error()
	}

	log.Printf("Role %v found\n", *role.RoleName)

	stsClient := sts.NewFromConfig(noPermsConfig)
	tempCredentials, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         role.Arn,
		RoleSessionName: aws.String("AssumeRoleExampleSession"),
		DurationSeconds: aws.Int32(900),
	})
	if err != nil {
		log.Printf("Couldn't assume role %v.\n", *role.RoleName)
		panic(err)
	}

	log.Printf("Assumed role Access Key ID: %v", *tempCredentials.Credentials.AccessKeyId)
	log.Printf("Assumed role Secret Access Key: %v", *tempCredentials.Credentials.SecretAccessKey)

	return "AssumeRole"
}

func GetKubernetesData(ctx context.Context) {
	kubeContext, kubeContextPresent := os.LookupEnv("KUBE_CONTEXT")
	if kubeContext == "" || !kubeContextPresent {
		log.Fatal("KUBE_CONTEXT is not set")
	}

	kubeConfigPath, kubeConfigPathPresent := os.LookupEnv("KUBECONFIG_PATH")
	if kubeConfigPath == "" || !kubeConfigPathPresent {
		log.Fatal("KUBECONFIG_PATH is not set")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	log.Printf("Kubeconfig built from %s", config.Host)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	namespaces := [4]string{"apps", "cluster-services", "monitoring", "datagovuk"}
	for _, namespace := range namespaces {
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Error listing pods in namespace %s: %s", namespace, err.Error())
		}
		log.Printf("There are %d pods in the %s namespace\n", len(pods.Items), namespace)
	}
}

func main() {
	ctx := context.TODO()
	AssumedRole(ctx)
	GetKubernetesData(ctx)
}