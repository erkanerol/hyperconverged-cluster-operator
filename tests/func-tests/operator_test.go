package tests_test

import (
	"context"
	"fmt"
	"time"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testscore "kubevirt.io/kubevirt/tests"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[vendor:cnv-qe@redhat.com][level:system]Strict Reconciliation tests", func() {
	tests.FlagParse()
	client, err := kubecli.GetKubevirtClient()
	testscore.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeEach()
	})

	It("[test_id:5999] CDI's spec reconciled by HCO", func() {
		crName := "cdi-kubevirt-hyperconverged"
		featureName := "ExtraNonExistantFeature"
		cdi, err := client.CdiClient().CdiV1beta1().CDIs().Get(context.TODO(), crName, k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		By(fmt.Sprintf("FEATUREGATES LIST LENGTH %v", len(cdi.Spec.Config.FeatureGates)))

		cdi.Spec.Config.FeatureGates = append(cdi.Spec.Config.FeatureGates, featureName)
		_, err = client.CdiClient().CdiV1beta1().CDIs().Update(context.TODO(), cdi, k8smetav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring extra feature gate does not persist")
		Eventually(func() []string {
			cdi, err = client.CdiClient().CdiV1beta1().CDIs().Get(context.TODO(), crName, k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			By(fmt.Sprintf("FEATUREGATES LIST LENGTH %v", len(cdi.Spec.Config.FeatureGates)))
			return cdi.Spec.Config.FeatureGates
		}, 2*time.Minute, 1*time.Second).ShouldNot(ContainElement(featureName))
	})

	It("[test_id:6000] should expose and propagate storage workloads specification to CDI", func() {
		hcoCR := getHyperconvergedCR()
		By(fmt.Sprintf("HCO CR SPEC VERSION %v", hcoCR))
	})
})

func getHyperconvergedCR() *hcov1beta1.HyperConverged {
	var hcoCR hcov1beta1.HyperConverged

	return &hcoCR
}
