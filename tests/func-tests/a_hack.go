package tests

import (
	"flag"
	_ "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"kubevirt.io/client-go/kubecli"
)

func init() {
	flag.Parse()
	kcFlag := flag.Lookup("kubeconfig")
	kubecli.SetKubeConfig(kcFlag.Value.String())
}
