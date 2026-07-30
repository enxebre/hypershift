package pki

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/openshift/hypershift/support/certs"
	"github.com/openshift/hypershift/support/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileAwsEbsCsiDriverControllerMetricsServingCertSecret(t *testing.T) {
	t.Parallel()

	caCfg := certs.CertCfg{IsCA: true, Subject: pkix.Name{CommonName: "root-ca", OrganizationalUnit: []string{"openshift"}}}
	caKey, caCert, err := certs.GenerateSelfSignedCertificate(&caCfg)
	if err != nil {
		t.Fatalf("failed to generate CA: %v", err)
	}
	caSecret := &corev1.Secret{
		Data: map[string][]byte{
			certs.CASignerCertMapKey: certs.CertToPem(caCert),
			certs.CASignerKeyMapKey:  certs.PrivateKeyToPem(caKey),
		},
	}

	ownerRef := config.OwnerRef{}

	tests := []struct {
		name      string
		namespace string
	}{
		{
			name:      "generates cert with correct DNS names",
			namespace: "clusters-my-hosted-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aws-ebs-csi-driver-controller-metrics-serving-cert",
					Namespace: tt.namespace,
				},
			}

			if err := ReconcileAwsEbsCsiDriverControllerMetricsServingCertSecret(secret, caSecret, ownerRef); err != nil {
				t.Fatalf("ReconcileAwsEbsCsiDriverControllerMetricsServingCertSecret returned error: %v", err)
			}

			if len(secret.Data[corev1.TLSCertKey]) == 0 {
				t.Error("expected TLS cert data to be populated")
			}
			if len(secret.Data[corev1.TLSPrivateKeyKey]) == 0 {
				t.Error("expected TLS key data to be populated")
			}

			certData, err := certs.PemToCertificate(secret.Data[corev1.TLSCertKey])
			if err != nil {
				t.Fatalf("failed to parse generated certificate: %v", err)
			}

			expectedDNSNames := []string{
				"aws-ebs-csi-driver-controller." + tt.namespace + ".svc",
				"aws-ebs-csi-driver-controller." + tt.namespace + ".svc.cluster.local",
				"aws-ebs-csi-driver-controller",
				"localhost",
			}

			if len(certData.DNSNames) != len(expectedDNSNames) {
				t.Errorf("expected %d DNS names, got %d: %v", len(expectedDNSNames), len(certData.DNSNames), certData.DNSNames)
			}
			for i, expected := range expectedDNSNames {
				if i >= len(certData.DNSNames) {
					break
				}
				if certData.DNSNames[i] != expected {
					t.Errorf("DNS name[%d]: expected %q, got %q", i, expected, certData.DNSNames[i])
				}
			}

			if certData.Subject.CommonName != "aws-ebs-csi-driver-controller" {
				t.Errorf("expected CN %q, got %q", "aws-ebs-csi-driver-controller", certData.Subject.CommonName)
			}

			if len(certData.Subject.Organization) != 1 || certData.Subject.Organization[0] != "openshift" {
				t.Errorf("expected organization [openshift], got %v", certData.Subject.Organization)
			}

			hasServerAuth := false
			hasClientAuth := false
			for _, usage := range certData.ExtKeyUsage {
				if usage == x509.ExtKeyUsageServerAuth {
					hasServerAuth = true
				}
				if usage == x509.ExtKeyUsageClientAuth {
					hasClientAuth = true
				}
			}
			if !hasServerAuth {
				t.Error("expected ExtKeyUsageServerAuth in certificate")
			}
			if !hasClientAuth {
				t.Error("expected ExtKeyUsageClientAuth in certificate")
			}
		})
	}
}
