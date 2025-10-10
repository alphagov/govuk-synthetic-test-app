package helpers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	stsTypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const CERT_PATH string = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
const INTEGRATION_AWS_ACCOUNT_ID string = "210287912431"
const STAGING_AWS_ACCOUNT_ID string = "696911096973"
const PRODUCTION_AWS_ACCOUNT_ID string = "172025368201"
const INTEGRATION string = "integration"
const STAGING string = "staging"
const PRODUCTION string = "production"
const CLUSTER_ID string = "govuk"
const REGION string = "eu-west-1"

func CheckRunningInK8s() (bool, error) {
	if _, err := os.Stat(CERT_PATH); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Can only run this code when inside a k8s pod\n")
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func GetEnvironmentAccountID(environment string) (string, error) {
	switch environment {
	case INTEGRATION:
		return INTEGRATION_AWS_ACCOUNT_ID, nil
	case STAGING:
		return STAGING_AWS_ACCOUNT_ID, nil
	case PRODUCTION:
		return PRODUCTION_AWS_ACCOUNT_ID, nil
	default:
		return INTEGRATION, fmt.Errorf("ENVIRONMENT_NAME not set to integration, staging or production")
	}
}

func GetK8sClient(ctx context.Context, environment string) (*http.Client, string, error) {
	running_in_k8s, err := CheckRunningInK8s()
	if err != nil {
		return nil, "", err
	} else if !running_in_k8s {
		return nil, "", nil
	}

	g, err := token.NewGenerator(false, false)
	if err != nil {
		return nil, "", err
	}

	environment_account_id, err := GetEnvironmentAccountID(environment)
	if err != nil {
		return nil, "", err
	}

	tk, err := g.GetWithOptions(ctx, &token.GetTokenOptions{
		Region:        REGION,
		ClusterID:     CLUSTER_ID,
		AssumeRoleARN: fmt.Sprintf("arn:aws:iam::%s:role/synthetic-test-assumed", environment_account_id),
		SessionName:   "GovUKSyntheticTestApp",
	})
	if err != nil {
		return nil, "", err
	}

	caCert, err := os.ReadFile(CERT_PATH)
	if err != nil {
		return nil, "", err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            caCertPool,
				InsecureSkipVerify: true,
			},
		},
	}

	return client, tk.Token, nil
}

func GetClusterEndpoint(ctx context.Context, environment string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(REGION))
	if err != nil {
		return "", err
	}
	sourceAccount := sts.NewFromConfig(cfg)

	environment_account_id, err := GetEnvironmentAccountID(environment)
	if err != nil {
		return "", err
	}

	rand.Seed(time.Now().UnixNano())
	response, err := sourceAccount.AssumeRole(
		ctx,
		&sts.AssumeRoleInput{
			RoleArn:         aws.String(fmt.Sprintf("arn:aws:iam::%s:role/synthetic-test-assumed", environment_account_id)),
			RoleSessionName: aws.String("GOVUK-Synthetic-Test-Assumed-" + strconv.Itoa(10000+rand.Intn(25000))),
		})
	if err != nil {
		return "", err
	}
	var assumedRoleCreds *stsTypes.Credentials = response.Credentials

	cfg, err = config.LoadDefaultConfig(
		ctx,
		config.WithRegion(REGION),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				*assumedRoleCreds.AccessKeyId,
				*assumedRoleCreds.SecretAccessKey,
				*assumedRoleCreds.SessionToken)))
	if err != nil {
		return "", err
	}

	eks_client := eks.NewFromConfig(cfg)

	cluster, err := eks_client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(CLUSTER_ID)})
	if err != nil {
		return "", err
	}

	return *cluster.Cluster.Endpoint, nil
}

func GetK8sAPIData(ctx context.Context, environment string, namespace string, resource_type string) ([]byte, error) {
	client, token, err := GetK8sClient(ctx, environment)
	if err != nil {
		return nil, err
	}

	cluster_endpoint, err := GetClusterEndpoint(ctx, environment)

	url, err := url.JoinPath(fmt.Sprintf("%s/api/v1/namespaces", cluster_endpoint), namespace, resource_type)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/yaml")

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("Error: %v, retrieving %v", err, url)
		return nil, err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("Error: %v, retrieving %v", err, url)
		return nil, err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error got %v status, retrieving %v", resp.StatusCode, url)
		return nil, err
	}

	return bodyText, nil
}

func GetPodList(ctx context.Context, environment string, namespace string) (*corev1.PodList, error) {
	bodyText_all, err := GetK8sAPIData(ctx, environment, namespace, "pods")
	if err != nil {
		return nil, err
	}

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime#Scheme
	scheme := runtime.NewScheme()

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/serializer#CodecFactory
	codecFactory := serializer.NewCodecFactory(scheme)

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime#Decoder
	deserializer := codecFactory.UniversalDeserializer()

	podObject, _, err := deserializer.Decode(bodyText_all, nil, &corev1.PodList{})
	if err != nil {
		return nil, err
	}
	podList := podObject.(*corev1.PodList)
	return podList, nil
}
