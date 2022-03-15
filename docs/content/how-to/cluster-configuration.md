# Configure OCP Components

## Overview

In standalone OpenShift, cluster configuration is achieved via cluster-scoped resources in the `config.openshift.io/v1`
API group. Resources such as APIServer, OAuth, and Proxy allow adding additional named certificates to the Kube APIServer, 
adding identity providers, configuring the global proxy, etc. In HyperShift, configuration resources that
impact the control plane need to be specified in the HostedCluster resource instead of inside the guest cluster. The
resources still exist inside the guest cluster, but their source of truth is the HostedCluster and are continuously
reconciled with what is specified in the HostedCluster.

## Configuration in a HostedCluster

The configuration resources that should be specified in the HostedCluster are:

* [APIServer](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/apiserver-config-openshift-io-v1.html) - Provides API server configuration such as certificates and certificate authorities.
* [Authentication](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/authentication-config-openshift-io-v1.html) - Controls the identity provider and authentication configuration for the cluster.
* [FeatureGate](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/featuregate-config-openshift-io-v1.html) - Enables FeatureGates so that you can use Tech Preview features.
* [Image](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/image-config-openshift-io-v1.html) - Configures how specific image registries should be treated (allowed, disallowed, insecure, CA details).
* [OAuth](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/oauth-config-openshift-io-v1.html) - Configures identity providers and other behavior related to internal OAuth server flows.
* [Proxy](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/proxy-config-openshift-io-v1.html) - Defines proxies to be used by components needing external network access. Note: not all components currently consume this value.
* [Scheduler](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/scheduler-config-openshift-io-v1.html) - Configures scheduler behavior such as policies and default node selectors.
* [Network](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/network-config-openshift-io-v1.html) - Configures network properties for initial cluster creation.

Resources that should still be configured inside the guest cluster are:

* [Build](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/build-config-openshift-io-v1.html) - Controls default and enforced configuration for all builds on the cluster.
* [Console](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/console-config-openshift-io-v1.html) - Configures the behavior of the web console interface, including the logout behavior.
* [Ingress](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/ingress-config-openshift-io-v1.html) - Configuration details related to routing such as the default domain for routes.
* [Project](https://docs.openshift.com/container-platform/4.9/rest_api/config_apis/project-config-openshift-io-v1.html) - Configures how projects are created including the project template.

## Including Configuration in a HostedCluster

Configuration resources should be specified inside the HostedCluster's `spec.configuration.items` field. This field contains
a list of desired configuration resources:

```
apiVersion: hypershift.openshift.io/v1alpha1
kind: HostedCluster
metadata:
  name: example
  namespace: clusters
spec:
  configuration:
    items:
    - apiVersion: config.openshift.io/v1
      kind: APIServer
      spec:
        ...
    - apiVersion: config.openshift.io/v1
      kind: OAuth
      spec:
        ...
    - apiVersion: config.openshift.io/v1
      kind: Proxy
      spec:
        ...
```

When embedding configuration, ensure that at the very least an `apiVersion` and a `kind` field are included for the
configuration item. Those fields are used to identify the resource that is being configured. Because these configuration
resources are singleton resources, no `metadata` needs to be specified for the resource.

## Referenced ConfigMaps and Secrets

Some configuration resources contain references to secrets or config maps. In standalone OpenShift, these are normally
expected in the `openshift-config` namespace. For HyperShift clusters, these need to be placed in the same namespace as
the `HostedCluster` resource and added to the list of references in either the `spec.configuration.secretRefs` or 
`spec.configuration.configMapRefs` field.

For example, when adding an additional serving certificate to the Kube APIServer, you would specify the serving certificate
secret as a reference in the `APIServer` resource. The same resource must also be included in the `spec.configuration.secretRefs`.
In the example below `my-serving-cert` is the name of a secret that is placed in the same namespace as the `HostedCluster`. 
It is referenced by both the `APIServer` configuration AND in the `secretRefs` list of secret resources.

```
apiVersion: hypershift.openshift.io/v1alpha1
kind: HostedCluster
metadata:
  name: example
  namespace: clusters
spec:
  configuration:
    items:
    - apiVersion: config.openshift.io/v1
      kind: APIServer
      spec:
        servingCerts:
          namedCertificates:
          - names:
            - xxx.example.com
            - yyy.example.com
            servingCertificate:
              name: my-serving-cert
  secretRefs:
  - name: my-serving-cert
```

## Configuration Validation

Configuration embedded in a HostedCluster is validated by HyperShift. If there are any issues reading the configuration
or there are secret or config map references missing, the `ValidConfiguration` condition in the `HostedCluster` status
will be set to `False` and the message will include information on the reason the configuration failed validation.

## OAuth configuration

The Oauth configuration section allows a user to specify the desired Oauth configuration for the internal Openshift Oauth server.
Guest cluster kube admin password will be exposed only when user has not explicitly specified the OAuth configuration. An example
configuration for an openID identity provider is shown below:
```
apiVersion: hypershift.openshift.io/v1alpha1
kind: HostedCluster
metadata:
  name: example
  namespace: master
spec:
  configuration:
    items:
    - apiVersion: config.openshift.io/v1
      kind: OAuth
      metadata:
        name: "example"
      spec:
        identityProviders:
        - openID:
            claims:
              email:
                - email
              name:
                - name
              preferredUsername:
                - preferred_username
            clientID: clientid1
            clientSecret:
              name: clientid1-secret-name
            issuer: https://example.com/identity
          mappingMethod: lookup
          name: IAM
          type: OpenID
    secretRefs:
    - name: "clientid1-secret-name"
```

For more details on the individual identity providers: refer to [upstream openshift documentation](https://docs.openshift.com/container-platform/4.9/authentication/understanding-identity-provider.html)