package cloud

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/openshift/backplane-cli/pkg/client/mocks"
	"github.com/openshift/backplane-cli/pkg/info"
	"github.com/openshift/backplane-cli/pkg/utils"
	mocks2 "github.com/openshift/backplane-cli/pkg/utils/mocks"
)

var _ = Describe("Cloud console command", func() {

	var (
		mockCtrl           *gomock.Controller
		mockClientWithResp *mocks.MockClientWithResponsesInterface
		mockOcmInterface   *mocks2.MockOCMInterface
		mockClientUtil     *mocks2.MockClientUtils

		proxyURI         string
		consoleAWSURL    string
		consoleGcloudURL string

		fakeAWSResp    *http.Response
		fakeGCloudResp *http.Response
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClientWithResp = mocks.NewMockClientWithResponsesInterface(mockCtrl)

		mockOcmInterface = mocks2.NewMockOCMInterface(mockCtrl)
		utils.DefaultOCMInterface = mockOcmInterface

		mockClientUtil = mocks2.NewMockClientUtils(mockCtrl)
		utils.DefaultClientUtils = mockClientUtil

		proxyURI = "https://shard.apps"
		consoleAWSURL = "https://signin.aws.amazon.com/federation?Action=login"
		consoleGcloudURL = "https://cloud.google.com/"

		mockClientWithResp.EXPECT().LoginClusterWithResponse(gomock.Any(), gomock.Any()).Return(nil, nil).Times(0)

		// Define fake AWS response
		fakeAWSResp = &http.Response{
			Body: MakeIoReader(
				fmt.Sprintf(`{"proxy_uri":"proxy", "message":"msg", "ConsoleLink":"%s"}`, consoleAWSURL),
			),
			Header:     map[string][]string{},
			StatusCode: http.StatusOK,
		}
		fakeAWSResp.Header.Add("Content-Type", "json")

		// Define fake AWS response
		fakeGCloudResp = &http.Response{
			Body: MakeIoReader(
				fmt.Sprintf(`{"proxy_uri":"proxy", "message":"msg", "ConsoleLink":"%s"}`, consoleGcloudURL),
			),
			Header:     map[string][]string{},
			StatusCode: http.StatusOK,
		}
		fakeGCloudResp.Header.Add("Content-Type", "json")

		// Clear config file
		_ = clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), api.Config{}, true)
		clientcmd.UseModifyConfigLock = false

		// Disabled log output
		log.SetOutput(io.Discard)
		os.Setenv(info.BackplaneURLEnvName, proxyURI)
	})

	AfterEach(func() {
		os.Setenv(info.BackplaneURLEnvName, "")
		mockCtrl.Finish()
	})
})
