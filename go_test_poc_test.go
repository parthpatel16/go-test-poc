package go_test_poc_test_test

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	certManagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmclientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = Describe("CertificateIssuance", func() {
	var (
		cmClient         *cmclientset.Clientset
		err              error
		certName         string = "test-certificate"
		defaultNamespace string = "testkube"
		secretName       string = "test-secret"
		IssuerName       string = "letsencrypt-dev-platformnintextestio"
		IssuerKind       string = "ClusterIssuer"
		certCommonName   string = "test.platform.nintextest.io"
		maxRetryAttempt  int    = 3
	)

	BeforeEach(func() {
		var config *rest.Config
		var err error
		var isRunningInCluster string = os.Getenv("RUNNING_IN_CLUSTER")

		if isRunningInCluster == "true" {
			fmt.Print("Using config from in-cluster\n")
			config, err = rest.InClusterConfig()
			Expect(err).NotTo(HaveOccurred(), "Should be able to create in-cluster config")

		} else {
			kubeconfigPath := os.Getenv("KUBECONFIG_PATH")

			if kubeconfigPath == "" {
				kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
			}

			fmt.Printf("Reading config from the path %s", kubeconfigPath)
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			Expect(err).NotTo(HaveOccurred(), "Should be able to build config from kubeconfig path")
		}

		cmClient, err = cmclientset.NewForConfig(config)
		Expect(err).NotTo(HaveOccurred(), "Should be able to create cert-manager client")
	})

	AfterEach(func() {
		err := cmClient.CertmanagerV1().Certificates(defaultNamespace).Delete(context.TODO(), certName, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("Error cleaning up certificate %s in namespace %s: %v\n", certName, defaultNamespace, err)
		}
	})

	It("should issue a certificate successfully", func() {
		cert := &certManagerv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certName,
				Namespace: defaultNamespace,
			},
			Spec: certManagerv1.CertificateSpec{
				SecretName: secretName,
				IssuerRef: cmmeta.ObjectReference{
					Name: IssuerName,
					Kind: IssuerKind,
				},
				CommonName: certCommonName,
				DNSNames:   []string{certCommonName},
			},
		}

		_, err = cmClient.CertmanagerV1().Certificates(defaultNamespace).Create(context.TODO(), cert, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Perform check multiple times if the Cert is not ready
		waitTime := 30 * time.Second
		for attempt := 1; attempt <= maxRetryAttempt; attempt++ {
			fmt.Printf("Checking if certificate is ready, attempt %d\n", attempt)

			// Fetch the Certificate to check its status
			issuedCert, err := cmClient.CertmanagerV1().Certificates(defaultNamespace).Get(context.TODO(), certName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to get the certificate")

			// Expect the Certificate to be ready
			certificateReady := false
			for _, condition := range issuedCert.Status.Conditions {
				if condition.Type == certManagerv1.CertificateConditionReady {
					Expect(condition.Type).To(BeEquivalentTo(certManagerv1.CertificateConditionReady))
					Expect(condition.Status).To(BeEquivalentTo(metav1.ConditionTrue))
					certificateReady = true
					break
				}
			}

			if certificateReady {
				return
			}

			if attempt < maxRetryAttempt {
				fmt.Printf("Certificate is not ready, wait for %v before retrying\n", waitTime)
				time.Sleep(waitTime)
				waitTime *= 2
			} else {
				Fail("Certificate did not become ready after %v attempts", maxRetryAttempt)
			}
		}
	})
})
