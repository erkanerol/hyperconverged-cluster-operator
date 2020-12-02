package operands

import (
	"errors"
	"reflect"

	networkaddonsshared "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/shared"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cnaHandler genericOperand

func newCnaHandler(Client client.Client, Scheme *runtime.Scheme) *cnaHandler {
	return &cnaHandler{
		Client: Client,
		Scheme: Scheme,
		crType: "NetworkAddonsConfig",
		isCr:   true,
		// Previous versions used to have HCO-operator (scope namespace)
		// as the owner of NetworkAddons (scope cluster).
		// It's not legal, so remove that.
		removeExistingOwner: true,
		hooks:               &cnaHooks{},
	}
}

type cnaHooks struct{}

func (h cnaHooks) getFullCr(hc *hcov1beta1.HyperConverged) runtime.Object {
	return NewNetworkAddons(hc)
}
func (h cnaHooks) getEmptyCr() runtime.Object                         { return &networkaddonsv1.NetworkAddonsConfig{} }
func (h cnaHooks) validate() error                                    { return nil }
func (h cnaHooks) postFound(*common.HcoRequest, runtime.Object) error { return nil }
func (h cnaHooks) getConditions(cr runtime.Object) []conditionsv1.Condition {
	return cr.(*networkaddonsv1.NetworkAddonsConfig).Status.Conditions
}
func (h cnaHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*networkaddonsv1.NetworkAddonsConfig)
	return checkComponentVersion(hcoutil.CnaoVersionEnvV, found.Status.ObservedVersion)
}
func (h cnaHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*networkaddonsv1.NetworkAddonsConfig).ObjectMeta
}

func (h *cnaHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	networkAddons, ok1 := required.(*networkaddonsv1.NetworkAddonsConfig)
	found, ok2 := exists.(*networkaddonsv1.NetworkAddonsConfig)

	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to CNA")
	}

	if !reflect.DeepEqual(found.Spec, networkAddons.Spec) && !req.UpgradeMode {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing Network Addons's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated Network Addons's Spec to its opinionated values")
		}
		networkAddons.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}

	return false, false, nil
}

func NewNetworkAddons(hc *hcov1beta1.HyperConverged, opts ...string) *networkaddonsv1.NetworkAddonsConfig {

	cnaoSpec := networkaddonsshared.NetworkAddonsConfigSpec{
		Multus:      &networkaddonsshared.Multus{},
		LinuxBridge: &networkaddonsshared.LinuxBridge{},
		Ovs:         &networkaddonsshared.Ovs{},
		NMState:     &networkaddonsshared.NMState{},
		KubeMacPool: &networkaddonsshared.KubeMacPool{},
	}

	cnaoInfra := hcoConfig2CnaoPlacement(hc.Spec.Infra.NodePlacement)
	cnaoWorkloads := hcoConfig2CnaoPlacement(hc.Spec.Workloads.NodePlacement)
	if cnaoInfra != nil || cnaoWorkloads != nil {
		cnaoSpec.PlacementConfiguration = &networkaddonsshared.PlacementConfiguration{
			Infra:     cnaoInfra,
			Workloads: cnaoWorkloads,
		}
	}

	if hc.Spec.CertConfig != nil {
		cnaoSpec.SelfSignConfiguration = &networkaddonsshared.SelfSignConfiguration{
			CARotateInterval:   hc.Spec.CertConfig.CARotateInterval.Duration.String(),
			CAOverlapInterval:  hc.Spec.CertConfig.CAOverlapInterval.Duration.String(),
			CertRotateInterval: hc.Spec.CertConfig.CertRotateInterval.Duration.String(),
		}
	}

	return &networkaddonsv1.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkaddonsnames.OPERATOR_CONFIG,
			Labels:    getLabels(hc),
			Namespace: getNamespace(hcoutil.UndefinedNamespace, opts),
		},
		Spec: cnaoSpec,
	}
}

func hcoConfig2CnaoPlacement(hcoConf *sdkapi.NodePlacement) *networkaddonsshared.Placement {
	if hcoConf == nil {
		return nil
	}
	empty := true
	cnaoPlacement := &networkaddonsshared.Placement{}
	if hcoConf.Affinity != nil {
		empty = false
		hcoConf.Affinity.DeepCopyInto(&cnaoPlacement.Affinity)
	}

	for _, hcoTol := range hcoConf.Tolerations {
		empty = false
		cnaoTol := corev1.Toleration{}
		hcoTol.DeepCopyInto(&cnaoTol)
		cnaoPlacement.Tolerations = append(cnaoPlacement.Tolerations, cnaoTol)
	}

	if len(hcoConf.NodeSelector) > 0 {
		empty = false
		cnaoPlacement.NodeSelector = make(map[string]string)
		for k, v := range hcoConf.NodeSelector {
			cnaoPlacement.NodeSelector[k] = v
		}
	}

	if empty {
		return nil
	}
	return cnaoPlacement
}