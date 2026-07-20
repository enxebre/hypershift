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

func TestReconcileAWSEBSCSIDriverControllerMetricsServingCertSecret(t *testing.T) {
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

	testCases := []struct {
		name   string
		secret *corev1.Secret
	}{
		{
			name: "When secret is empty, it should populate TLS cert and key with correct DNS names",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "aws-ebs-csi-driver-controller-metrics-serving-cert",
					Namespace: "test-namespace",
				},
			},
		},
		{
			name: "When secret already has valid cert data, it should remain stable on re-reconciliation",
			secret: func() *corev1.Secret {
				s := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-ebs-csi-driver-controller-metrics-serving-cert",
						Namespace: "test-namespace",
					},
				}
				if err := ReconcileAWSEBSCSIDriverControllerMetricsServingCertSecret(s, caSecret, ownerRef); err != nil {
					t.Fatalf("failed to pre-populate secret: %v", err)
				}
				return s
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ReconcileAWSEBSCSIDriverControllerMetricsServingCertSecret(tc.secret, caSecret, ownerRef); err != nil {
				t.Fatalf("ReconcileAWSEBSCSIDriverControllerMetricsServingCertSecret failed: %v", err)
			}

			if len(tc.secret.Data[corev1.TLSCertKey]) == 0 {
				t.Error("expected TLS cert to be populated")
			}
			if len(tc.secret.Data[corev1.TLSPrivateKeyKey]) == 0 {
				t.Error("expected TLS private key to be populated")
			}

			certData, err := certs.PemToCertificate(tc.secret.Data[corev1.TLSCertKey])
			if err != nil {
				t.Fatalf("failed to parse generated cert: %v", err)
			}

			expectedDNSNames := []string{
				"aws-ebs-csi-driver-controller.test-namespace.svc",
				"aws-ebs-csi-driver-controller.test-namespace.svc.cluster.local",
				"aws-ebs-csi-driver-controller",
				"localhost",
			}

			if len(certData.DNSNames) != len(expectedDNSNames) {
				t.Errorf("expected %d DNS names, got %d: %v", len(expectedDNSNames), len(certData.DNSNames), certData.DNSNames)
			}

			dnsNameSet := map[string]bool{}
			for _, name := range certData.DNSNames {
				dnsNameSet[name] = true
			}
			for _, expected := range expectedDNSNames {
				if !dnsNameSet[expected] {
					t.Errorf("expected DNS name %q not found in cert", expected)
				}
			}

			if certData.Subject.CommonName != "aws-ebs-csi-driver-controller" {
				t.Errorf("expected common name %q, got %q", "aws-ebs-csi-driver-controller", certData.Subject.CommonName)
			}

			if len(certData.Subject.Organization) != 1 || certData.Subject.Organization[0] != "openshift" {
				t.Errorf("expected organization [openshift], got %v", certData.Subject.Organization)
			}

			if certData.KeyUsage&x509.KeyUsageKeyEncipherment == 0 || certData.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
				t.Error("expected key usage to include key encipherment and digital signature")
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
			if !hasServerAuth || !hasClientAuth {
				t.Errorf("expected ExtKeyUsage to include both server and client auth, got %v", certData.ExtKeyUsage)
			}
		})
	}
}
