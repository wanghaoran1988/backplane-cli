package cloud

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/backplane-cli/pkg/cli/config"
	"github.com/openshift/backplane-cli/pkg/utils"
	mocks2 "github.com/openshift/backplane-cli/pkg/utils/mocks"
)

//nolint:gosec
var _ = Describe("getIsolatedCredentials", func() {
	var (
		mockCtrl         *gomock.Controller
		mockOcmInterface *mocks2.MockOCMInterface
		mockClientUtil   *mocks2.MockClientUtils

		testOcmToken        string
		testClusterID       string
		testAccessKeyID     string
		testSecretAccessKey string
		testSessionToken    string
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())

		mockOcmInterface = mocks2.NewMockOCMInterface(mockCtrl)
		utils.DefaultOCMInterface = mockOcmInterface

		mockClientUtil = mocks2.NewMockClientUtils(mockCtrl)
		utils.DefaultClientUtils = mockClientUtil

		testOcmToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiZW1haWwiOiJ0ZXN0QGZvby5jb20iLCJpYXQiOjE1MTYyMzkwMjJ9.5NG4wFhitEKZyzftSwU67kx4JVTEWcEoKhCl_AFp8T4"
		testClusterID = "test123"
		testAccessKeyID = "test-access-key-id"
		testSecretAccessKey = "test-secret-access-key"
		testSessionToken = "test-session-token"
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Execute getIsolatedCredentials", func() {
		It("should fail if no argument is provided", func() {
			_, err := getIsolatedCredentials("")
			Expect(err).To(Equal(fmt.Errorf("must provide non-empty cluster ID")))
		})
		It("should fail if cannot retrieve OCM token", func() {
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(nil, errors.New("foo")).Times(1)

			_, err := getIsolatedCredentials(testClusterID)
			Expect(err.Error()).To(Equal("failed to retrieve OCM token: foo"))
		})
		It("should fail if cannot retrieve backplane configuration", func() {
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testOcmToken, nil).Times(1)
			GetBackplaneConfiguration = func() (bpConfig config.BackplaneConfiguration, err error) {
				return config.BackplaneConfiguration{}, errors.New("oops")
			}

			_, err := getIsolatedCredentials(testClusterID)
			Expect(err.Error()).To(Equal("error retrieving backplane configuration: oops"))
		})
		It("should fail if backplane configuration does not contain value for AssumeInitialArn", func() {
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testOcmToken, nil).Times(1)
			GetBackplaneConfiguration = func() (bpConfig config.BackplaneConfiguration, err error) {
				return config.BackplaneConfiguration{
					URL:              "testUrl.com",
					ProxyURL:         "testProxyUrl.com",
					AssumeInitialArn: "",
				}, nil
			}

			_, err := getIsolatedCredentials(testClusterID)
			Expect(err.Error()).To(Equal("backplane config is missing required `assume-initial-arn` property"))
		})
		It("should fail if cannot create sts client with proxy", func() {
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testOcmToken, nil).Times(1)
			GetBackplaneConfiguration = func() (bpConfig config.BackplaneConfiguration, err error) {
				return config.BackplaneConfiguration{
					URL:              "testUrl.com",
					ProxyURL:         "testProxyUrl.com",
					AssumeInitialArn: "arn:aws:iam::123456789:role/ManagedOpenShift-Support-Role",
				}, nil
			}
			StsClientWithProxy = func(proxyURL string) (*sts.Client, error) {
				return nil, errors.New(":(")
			}

			_, err := getIsolatedCredentials(testClusterID)
			Expect(err.Error()).To(Equal("failed to create sts client: :("))
		})
		It("should fail if initial role cannot be assumed with JWT", func() {
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testOcmToken, nil).Times(1)
			GetBackplaneConfiguration = func() (bpConfig config.BackplaneConfiguration, err error) {
				return config.BackplaneConfiguration{
					URL:              "testUrl.com",
					ProxyURL:         "testProxyUrl.com",
					AssumeInitialArn: "arn:aws:iam::123456789:role/ManagedOpenShift-Support-Role",
				}, nil
			}
			StsClientWithProxy = func(proxyURL string) (*sts.Client, error) {
				return &sts.Client{}, nil
			}
			AssumeRoleWithJWT = func(jwt string, roleArn string, stsClient stscreds.AssumeRoleWithWebIdentityAPIClient) (aws.Credentials, error) {
				return aws.Credentials{}, errors.New("failure")
			}

			_, err := getIsolatedCredentials(testClusterID)
			Expect(err.Error()).To(Equal("failed to assume role using JWT: failure"))
		})
		It("should fail if email cannot be pulled off JWT", func() {
			testOcmToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testOcmToken, nil).Times(1)
			GetBackplaneConfiguration = func() (bpConfig config.BackplaneConfiguration, err error) {
				return config.BackplaneConfiguration{
					URL:              "testUrl.com",
					ProxyURL:         "testProxyUrl.com",
					AssumeInitialArn: "arn:aws:iam::123456789:role/ManagedOpenShift-Support-Role",
				}, nil
			}
			StsClientWithProxy = func(proxyURL string) (*sts.Client, error) {
				return &sts.Client{}, nil
			}
			AssumeRoleWithJWT = func(jwt string, roleArn string, stsClient stscreds.AssumeRoleWithWebIdentityAPIClient) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     testAccessKeyID,
					SecretAccessKey: testSecretAccessKey,
					SessionToken:    testSessionToken,
				}, nil
			}

			_, err := getIsolatedCredentials(testClusterID)
			Expect(err.Error()).To(Equal("unable to extract email from given token: no field email on given token"))
		})
		It("should fail if error creating backplane api client", func() {
			mockOcmInterface.EXPECT().GetOCMAccessToken().Return(&testOcmToken, nil).Times(1)
			GetBackplaneConfiguration = func() (bpConfig config.BackplaneConfiguration, err error) {
				return config.BackplaneConfiguration{
					URL:              "testUrl.com",
					ProxyURL:         "testProxyUrl.com",
					AssumeInitialArn: "arn:aws:iam::123456789:role/ManagedOpenShift-Support-Role",
				}, nil
			}
			StsClientWithProxy = func(proxyURL string) (*sts.Client, error) {
				return &sts.Client{}, nil
			}
			AssumeRoleWithJWT = func(jwt string, roleArn string, stsClient stscreds.AssumeRoleWithWebIdentityAPIClient) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     testAccessKeyID,
					SecretAccessKey: testSecretAccessKey,
					SessionToken:    testSessionToken,
				}, nil
			}
			NewStaticCredentialsProvider = func(key, secret, session string) credentials.StaticCredentialsProvider {
				return credentials.StaticCredentialsProvider{}
			}
			mockClientUtil.EXPECT().MakeRawBackplaneAPIClientWithAccessToken("testUrl.com", testOcmToken).Return(nil, errors.New("foo")).Times(1)

			_, err := getIsolatedCredentials(testClusterID)
			Expect(err.Error()).To(Equal("failed to create backplane client with access token: foo"))
		})
	})
})

// newTestCluster assembles a *cmv1.Cluster while handling the error to help out with inline test-case generation
func newTestCluster(t *testing.T, cb *cmv1.ClusterBuilder) *cmv1.Cluster {
	cluster, err := cb.Build()
	if err != nil {
		t.Fatalf("failed to build cluster: %s", err)
	}

	return cluster
}

func TestIsIsolatedBackplaneAccess(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *cmv1.Cluster
		expected bool
	}{
		{
			name:     "AWS non-STS",
			cluster:  newTestCluster(t, cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().Enabled(false)))),
			expected: false,
		},
		{
			name:     "GCP",
			cluster:  newTestCluster(t, cmv1.NewCluster().GCP(cmv1.NewGCP())),
			expected: false,
		},
	}

	//cmv1.NewStsSupportJumpRole().RoleArn(OldFlowSupportRole)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := isIsolatedBackplaneAccess(test.cluster)
			if err != nil {
				t.Errorf("unexpected err: %v", err)
			}

			if test.expected != actual {
				t.Errorf("expected: %v, got: %v", test.expected, actual)
			}
		})
	}
}

var _ = Describe("isIsolatedBackplaneAccess", func() {
	var (
		mockCtrl         *gomock.Controller
		mockOcmInterface *mocks2.MockOCMInterface

		testClusterID string
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())

		mockOcmInterface = mocks2.NewMockOCMInterface(mockCtrl)
		utils.DefaultOCMInterface = mockOcmInterface

		testClusterID = "test123"
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Execute isIsolatedBackplaneAccess", func() {
		It("returns an error if fails to get STS Support Jump Role from OCM for STS enabled cluster", func() {
			mockOcmInterface.EXPECT().GetStsSupportJumpRoleARN(testClusterID).Return("", errors.New("oops"))

			stsBuilder := &cmv1.STSBuilder{}
			stsBuilder.Enabled(true)

			awsBuilder := &cmv1.AWSBuilder{}
			awsBuilder.STS(stsBuilder)

			clusterBuilder := cmv1.ClusterBuilder{}
			clusterBuilder.AWS(awsBuilder)
			clusterBuilder.ID(testClusterID)

			cluster, _ := clusterBuilder.Build()
			_, err := isIsolatedBackplaneAccess(cluster)

			Expect(err).NotTo(BeNil())
		})
		It("returns an error if fails to parse STS Support Jump Role from OCM for STS enabled cluster", func() {
			mockOcmInterface.EXPECT().GetStsSupportJumpRoleARN(testClusterID).Return("not-an-arn", nil)

			stsBuilder := &cmv1.STSBuilder{}
			stsBuilder.Enabled(true)

			awsBuilder := &cmv1.AWSBuilder{}
			awsBuilder.STS(stsBuilder)

			clusterBuilder := cmv1.ClusterBuilder{}
			clusterBuilder.AWS(awsBuilder)
			clusterBuilder.ID(testClusterID)

			cluster, _ := clusterBuilder.Build()
			_, err := isIsolatedBackplaneAccess(cluster)

			Expect(err).NotTo(BeNil())
		})
		It("returns false with no error for STS enabled cluster with ARN that matches old support flow ARN", func() {
			mockOcmInterface.EXPECT().GetStsSupportJumpRoleARN(testClusterID).Return("arn:aws:iam::123456789:role/RH-Technical-Support-Access", nil)

			stsBuilder := &cmv1.STSBuilder{}
			stsBuilder.Enabled(true)

			awsBuilder := &cmv1.AWSBuilder{}
			awsBuilder.STS(stsBuilder)

			clusterBuilder := cmv1.ClusterBuilder{}
			clusterBuilder.AWS(awsBuilder)
			clusterBuilder.ID(testClusterID)

			cluster, _ := clusterBuilder.Build()
			result, err := isIsolatedBackplaneAccess(cluster)

			Expect(result).To(Equal(false))
			Expect(err).To(BeNil())
		})
		It("returns true with no error for STS enabled cluster with ARN that doesn't match old support flow ARN", func() {
			mockOcmInterface.EXPECT().GetStsSupportJumpRoleARN(testClusterID).Return("arn:aws:iam::123456789:role/RH-Technical-Support-12345", nil)

			stsBuilder := &cmv1.STSBuilder{}
			stsBuilder.Enabled(true)

			awsBuilder := &cmv1.AWSBuilder{}
			awsBuilder.STS(stsBuilder)

			clusterBuilder := cmv1.ClusterBuilder{}
			clusterBuilder.AWS(awsBuilder)
			clusterBuilder.ID(testClusterID)

			cluster, _ := clusterBuilder.Build()
			result, err := isIsolatedBackplaneAccess(cluster)

			Expect(result).To(Equal(true))
			Expect(err).To(BeNil())
		})
	})
})
