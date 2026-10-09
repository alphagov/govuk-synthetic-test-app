package helpers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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

const (
	CERT_PATH                  string = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	INTEGRATION_AWS_ACCOUNT_ID string = "210287912431"
	STAGING_AWS_ACCOUNT_ID     string = "696911096973"
	PRODUCTION_AWS_ACCOUNT_ID  string = "172025368201"
	CLUSTER_ID                 string = "govuk"
	REGION                     string = "eu-west-1"
	ASSUME_ROLE_NAME           string = "synthetic-test-assumer"
)

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

type K8sClient struct {
	Client          *http.Client
	Token           string
	ClusterEndpoint string
}

func (k *K8sClient) Get(ctx context.Context, url string) (*http.Response, error) {
	fullURL := strings.TrimPrefix(url, "/")
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", k.ClusterEndpoint, fullURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+k.Token)
	req.Header.Set("Accept", "application/json")

	req = req.WithContext(ctx)

	return k.Client.Do(req)
}

func GetAwsAccountID(ctx context.Context) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(REGION))
	if err != nil {
		return "", err
	}
	sourceAccount := sts.NewFromConfig(cfg)

	callerIdentity, err := sourceAccount.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *callerIdentity.Account, nil
}

func GetK8sClient(ctx context.Context, accountID string, clusterID string, roleName string) (*K8sClient, error) {
	runningInK8s, err := CheckRunningInK8s()
	if err != nil {
		return nil, err
	} else if !runningInK8s {
		return nil, nil
	}

	assumeRoleARN := fmt.Sprintf("arn:aws:iam::%s:role/synthetic-test-assumed", accountID)

	g, err := token.NewGenerator(false, false)
	if err != nil {
		return nil, err
	}

	tk, err := g.GetWithOptions(ctx, &token.GetTokenOptions{
		Region:        REGION,
		ClusterID:     clusterID,
		AssumeRoleARN: assumeRoleARN,
		SessionName:   "GovUKSyntheticTestApp",
	})
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(REGION))
	if err != nil {
		return nil, err
	}

	sourceAccount := sts.NewFromConfig(cfg)

	rand.Seed(time.Now().UnixNano())
	response, err := sourceAccount.AssumeRole(
		ctx,
		&sts.AssumeRoleInput{
			RoleArn:         aws.String(assumeRoleARN),
			RoleSessionName: aws.String("GOVUK-Synthetic-Test-Assumed-" + strconv.Itoa(10000+rand.Intn(25000))),
		})
	if err != nil {
		return nil, err
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
		return nil, err
	}

	eks_client := eks.NewFromConfig(cfg)

	cluster, err := eks_client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(clusterID)})
	if err != nil {
		return nil, err
	}

	caCert, err := base64.StdEncoding.DecodeString(*cluster.Cluster.CertificateAuthority.Data)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	return &K8sClient{
		Client:          client,
		Token:           tk.Token,
		ClusterEndpoint: *cluster.Cluster.Endpoint,
	}, nil
}

func GetK8sAPIData(ctx context.Context, accountID string, clusterID string, roleName string, namespace string, resource_type string) ([]byte, error) {
	client, err := GetK8sClient(ctx, accountID, clusterID, roleName)
	if err != nil {
		return nil, err
	}
	url, err := url.JoinPath("api", "v1", "namespaces", namespace, resource_type)
	if err != nil {
		return nil, err
	}

	resp, err := client.Get(ctx, url)
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

func GetPodList(ctx context.Context, accountID string, clusterID string, roleName string, namespace string) (*corev1.PodList, error) {
	bodyText_all, err := GetK8sAPIData(ctx, accountID, clusterID, roleName, namespace, "pods")
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	codecFactory := serializer.NewCodecFactory(scheme)

	deserializer := codecFactory.UniversalDeserializer()

	podObject, _, err := deserializer.Decode(bodyText_all, nil, &corev1.PodList{})
	if err != nil {
		return nil, err
	}
	podList := podObject.(*corev1.PodList)
	return podList, nil
}
