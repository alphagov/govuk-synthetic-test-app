package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type RoleData struct {
	RoleName string
	Arn      string
}

func AssumedRole(ctx context.Context) string {
	noPermsConfig, err := config.LoadDefaultConfig(ctx)

	role := RoleData{
		RoleName: "synthetic-test-assumed",
		Arn:      "arn:aws:iam::210287912431:role/synthetic-test-assumed",
	}

	stsClient := sts.NewFromConfig(noPermsConfig)
	tempCredentials, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(role.Arn),
		RoleSessionName: aws.String("AssumeRoleExampleSession"),
		DurationSeconds: aws.Int32(900),
	})
	if err != nil {
		log.Printf("Couldn't assume role %v.\n", role.RoleName)
		panic(err)
	}

	log.Printf("Assumed role Access Key ID: %v", *tempCredentials.Credentials.AccessKeyId)
	log.Printf("Assumed role Secret Access Key: %v", *tempCredentials.Credentials.SecretAccessKey)

	return "AssumeRole"
}

func GetKubernetesData(ctx context.Context) {
	var config *rest.Config
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else if os.IsNotExist(err) {
		kubeConfigPath, kubeConfigPathPresent := os.LookupEnv("KUBECONFIG_PATH")
		if kubeConfigPath == "" || !kubeConfigPathPresent {
			log.Fatal("KUBECONFIG_PATH is not set")
		}
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			panic("No serviceaccount mounted or -kubeconfig flag passed or .kube/config file \n " + err.Error())
		}
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	AssumedRole(ctx)

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
	GetKubernetesData(ctx)
}
