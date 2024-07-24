package karpenter

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift/hypershift/control-plane-operator/hostedclusterconfigoperator/controllers/featuregate/karpenter/assets"
	supportassets "github.com/openshift/hypershift/support/assets"
	"github.com/openshift/hypershift/support/upsert"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	crdEC2NodeClass = supportassets.MustCRD(assets.ReadFile, "karpenter.k8s.aws_ec2nodeclasses.yaml")
	crdNodePool     = supportassets.MustCRD(assets.ReadFile, "karpenter.sh_nodepools.yaml")
	crdNodeClaim    = supportassets.MustCRD(assets.ReadFile, "karpenter.sh_nodeclaims.yaml")
)

// Reconciler does the following:
// Reconcile Karpeneter CRDs.
// Approves any CSRs.
// Reconcile Instances of ec2NodeClass with known userdata.
type Reconciler struct {
	Client             client.Client
	GuestClusterClient client.Client
	upsert.CreateOrUpdateProvider
	HCPNamespace string
}

func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling")

	// Reconcile CRDs.
	{
		errs := []error{}
		for _, crd := range []*apiextensionsv1.CustomResourceDefinition{
			crdEC2NodeClass,
			crdNodePool,
			crdNodeClaim,
		} {
			_, err := r.CreateOrUpdate(ctx, r.GuestClusterClient, crd, func() error {
				return nil
			})
			if err != nil {
				errs = append(errs, err)
			}
		}
		if err := utilerrors.NewAggregate(errs); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to reconcile CRDs: %w", err)
		}
	}

	// Approve all CSRs.
	{
		csrList := &certv1.CertificateSigningRequestList{}
		err := r.GuestClusterClient.List(ctx, csrList)
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}

		for _, csr := range csrList.Items {
			for _, condition := range csr.Status.Conditions {
				if condition.Type == certv1.CertificateApproved {
					break
				}
			}

			csr.Status.Conditions = append(csr.Status.Conditions, certv1.CertificateSigningRequestCondition{
				Type:           certv1.CertificateApproved,
				Status:         "True",
				Reason:         "AutoApproved",
				Message:        "Approved by hypershift feature gate Karpeneter Controller",
				LastUpdateTime: metav1.Now(),
			})
			err = r.GuestClusterClient.Status().Update(ctx, &csr)
			if err != nil {
				return ctrl.Result{}, err
			}
			log.Info("Approved CSR", "csr", csr.Name)
		}
	}

	// Inject userData so we keep the ignition authorization token up todate in the ec2Class.
	// TODO(alberto): Reconcile the amiID as well.
	{
		labelSelector := labels.SelectorFromSet(labels.Set{"featuregate": "autoprovision"})
		listOptions := &client.ListOptions{
			LabelSelector: labelSelector,
			Namespace:     r.HCPNamespace,
		}
		secretList := &corev1.SecretList{}
		err := r.Client.List(ctx, secretList, listOptions)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to list secrets: %w", err)
		}
		if len(secretList.Items) != 1 {
			return ctrl.Result{}, fmt.Errorf("expected 1 secret, got %d", len(secretList.Items))
		}
		secret := secretList.Items[0]

		ec2NodeClassList := &unstructured.UnstructuredList{}
		ec2NodeClassList.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "karpenter.k8s.aws",
			Version: "v1beta1",
			Kind:    "EC2NodeClassList",
		})
		err = r.GuestClusterClient.List(ctx, ec2NodeClassList)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get EC2NodeClassList: %w", err)
		}

		errs := []error{}
		for _, ec2NodeClass := range ec2NodeClassList.Items {
			ec2NodeClass.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "karpenter.k8s.aws",
				Version: "v1beta1",
				Kind:    "EC2NodeClass",
			})
			_, err = r.CreateOrUpdate(ctx, r.GuestClusterClient, &ec2NodeClass, func() error {
				ec2NodeClass.Object["spec"].(map[string]interface{})["userData"] = string(secret.Data["value"])
				return nil
			})
			if err != nil {
				errs = append(errs, err)
			}
			if err == nil {
				log.Info("Set userData in ec2NodeClass", "ec2NodeClass", ec2NodeClass.GetName())
			}
		}
		if err := utilerrors.NewAggregate(errs); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update EC2NodeClass: %w", err)
		}
	}

	return reconcile.Result{
		Requeue: false,
		// Requeue every 1 seconds to go over EC2NodeClassList so we don't have to vendor and watch types.
		// TODO(alberto): make this event driven.
		RequeueAfter: 1 * time.Second,
	}, nil
}
