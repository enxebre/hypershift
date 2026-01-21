# Spot Instance Support Implementation Tracker

## Overview
This document tracks the implementation of spot instance support for HyperShift NodePools with proper interruption handling, automatic node draining, and machine health checking.

---

## Phase 1: NodePool Controller Changes

### Task 1.1: Create annotation-based spot enablement function
- [ ] Add constant `AnnotationEnableSpot = "hypershift.openshift.io/enable-spot"` to [hypershift-operator/controllers/nodepool/aws.go](hypershift-operator/controllers/nodepool/aws.go)
- [ ] Implement `isSpotEnabled(nodePool)` function
- [ ] Add unit test `TestIsSpotEnabled()` covering annotation present/absent/wrong value

**File**: hypershift-operator/controllers/nodepool/aws.go

---

### Task 1.2: Add spot configuration to AWSMachineTemplate
- [ ] Modify `awsMachineTemplateSpec()` function
- [ ] Add logic to check `isSpotEnabled(nodePool)`
- [ ] Set `SpotMarketOptions` with nil MaxPrice when spot enabled
- [ ] Add spot instance tags including `aws-node-termination-handler/managed`
- [ ] Add unit test `TestAWSMachineTemplateWithSpot()`

**File**: hypershift-operator/controllers/nodepool/aws.go

---

### Task 1.3: Add termination handler tag to spot instances
- [ ] Modify `awsAdditionalTags()` function
- [ ] Add tag `aws-node-termination-handler/managed: ""` when spot is enabled
- [ ] Add unit test `TestAWSTagsWithTerminationHandler()`

**File**: hypershift-operator/controllers/nodepool/aws.go

---

### Task 1.4: Add interruptible-instance node label
- [ ] Modify `reconcileMachineDeployment()` function
- [ ] Modify `reconcileMachineSet()` function
- [ ] Add label `machine.openshift.io/interruptible-instance: ""` when spot enabled

**File**: hypershift-operator/controllers/nodepool/capi.go

---

## Phase 2: Control Plane Operator v2 Component

### Task 2.1: Create aws-node-termination-handler component structure
- [ ] Create directory: control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/
- [ ] Create component.go with:
  - [ ] ComponentName constant
  - [ ] options struct with ComponentOptions interface implementation
  - [ ] NewComponent() function
  - [ ] AnnotationTerminationHandlerQueueURL constant
  - [ ] predicate() function checking AWS platform and queue URL annotation
- [ ] Create deployment.go file
- [ ] Create secret.go file

**Files**:
- control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/component.go
- control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/deployment.go
- control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/secret.go

---

### Task 2.2: Create deployment manifest asset
- [ ] Create directory: control-plane-operator/controllers/hostedcontrolplane/v2/assets/aws-node-termination-handler/
- [ ] Create deployment.yaml with:
  - [ ] 1 replica Deployment
  - [ ] aws-node-termination-handler container with placeholder image
  - [ ] Volume mounts: kubeconfig, credentials, token volumes
  - [ ] Placeholder environment variables
  - [ ] Resource limits: cpu 100m, memory 128Mi

**File**: control-plane-operator/controllers/hostedcontrolplane/v2/assets/aws-node-termination-handler/deployment.yaml

---

### Task 2.3: Implement deployment adapter
- [ ] Implement `adaptDeployment()` function setting:
  - [ ] AWS_REGION from HCP.Spec.Platform.AWS.Region
  - [ ] QUEUE_URL from HCP annotation
  - [ ] KUBERNETES_SERVICE_HOST/PORT from HCP status
  - [ ] ENABLE_SQS_TERMINATION_DRAINING=true
  - [ ] ENABLE_SPOT_INTERRUPTION_DRAINING=true
  - [ ] Namespace references
- [ ] Add unit test `TestDeploymentAdapter()`

**File**: control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/deployment.go

---

### Task 2.4: Create AWS credentials secret manifest and adapter
- [ ] Create credentials-secret.yaml template
- [ ] Implement `adaptCredentialsSecret()` function
- [ ] Populate secret with AWS credentials from HCP platform credentials
- [ ] Verify secret format is valid

**Files**:
- control-plane-operator/controllers/hostedcontrolplane/v2/assets/aws-node-termination-handler/credentials-secret.yaml
- control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/secret.go

---

### Task 2.5: Register component in hostedcontrolplane controller
- [ ] Add import for awsnodeterminationhandler package
- [ ] Register component in components slice around line 255
- [ ] Verify component appears in ControlPlaneComponent CRD status

**File**: control-plane-operator/controllers/hostedcontrolplane/hostedcontrolplane_controller.go

---

## Phase 3: MachineHealthCheck Deployment

### Task 3.1: Create MachineHealthCheck manifest asset
- [ ] Create machinehealthcheck.yaml with:
  - [ ] CAPI MachineHealthCheck apiVersion
  - [ ] Selector matching `machine.openshift.io/interruptible-instance: ""`
  - [ ] unhealthyConditions for Ready=False/Unknown with 300s timeout
  - [ ] maxUnhealthy: "100%"
  - [ ] nodeStartupTimeout: "10m"

**File**: control-plane-operator/controllers/hostedcontrolplane/v2/assets/aws-node-termination-handler/machinehealthcheck.yaml

---

### Task 3.2: Implement MachineHealthCheck adapter
- [ ] Create machinehealthcheck.go
- [ ] Implement `adaptMachineHealthCheck()` function setting:
  - [ ] Namespace = HCP.Namespace
  - [ ] ClusterName = HCP infrastructure ID
  - [ ] Validate selector label matches NodePool label
- [ ] Add unit test `TestMachineHealthCheckAdapter()`

**File**: control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/machinehealthcheck.go

---

### Task 3.3: Register MachineHealthCheck in component
- [ ] Add `.WithManifestAdapter("machinehealthcheck.yaml", ...)` to component builder
- [ ] Verify MachineHealthCheck is created alongside Deployment

**File**: control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/component.go

---

## Phase 4: E2E Testing Infrastructure

### Task 4.1: Create spot NodePool E2E test structure
- [ ] Create nodepool_spot_test.go
- [ ] Implement spotNodePoolTest struct with:
  - [ ] DummyInfraSetup
  - [ ] ctx, ec2Client, hostedClusterClient, hostedCluster, sqsQueueURL fields
- [ ] Implement Setup() method
- [ ] Implement BuildNodePoolManifest() method
- [ ] Implement Run() method
- [ ] Verify test structure compiles

**File**: test/e2e/nodepool_spot_test.go

---

### Task 4.2: Implement spot instance validation test
- [ ] Create TestSpotInstanceCreation test function
- [ ] Create NodePool with annotation `hypershift.openshift.io/enable-spot: "true"`
- [ ] Wait for nodes using WaitForReadyNodesByNodePool()
- [ ] Extract instance IDs from node ProviderID fields
- [ ] Query EC2 API to validate:
  - [ ] Instances are spot instances (InstanceLifecycle == "spot")
  - [ ] Tag `aws-node-termination-handler/managed: ""` exists
- [ ] Validate nodes have label `machine.openshift.io/interruptible-instance: ""`

**File**: test/e2e/nodepool_spot_test.go

---

### Task 4.3: Implement spot interruption simulation test
- [ ] Create nodepool_spot_interruption_test.go
- [ ] Create TestSpotInterruptionHandling test function
- [ ] Create spot NodePool and wait for ready nodes
- [ ] Pick one node/instance
- [ ] Send mock spot interruption warning to SQS queue
- [ ] Validate node draining:
  - [ ] Node becomes cordoned
  - [ ] Pods are evicted gracefully
  - [ ] Termination handler logs show drain completion
- [ ] Terminate instance via EC2 API
- [ ] Validate MachineHealthCheck response:
  - [ ] Machine transitions to unhealthy
  - [ ] Machine is deleted
  - [ ] New Machine is created
- [ ] Wait for new node to become Ready
- [ ] Validate NodePool maintains replica count

**File**: test/e2e/nodepool_spot_interruption_test.go

---

### Task 4.4: Create test utilities for spot instances
- [ ] Create test/e2e/util/spot.go
- [ ] Implement IsSpotInstance()
- [ ] Implement GetInstanceTags()
- [ ] Implement SendSpotInterruptionWarning()
- [ ] Implement WaitForNodeCordon()
- [ ] Implement WaitForPodsEvicted()
- [ ] Add unit tests for each utility function

**File**: test/e2e/util/spot.go

---

## Phase 5: Integration & Documentation

### Task 5.1: Add go.mod dependencies
- [ ] Verify AWS SDK dependencies for SQS are present
- [ ] Run `go mod tidy`
- [ ] Run `go mod vendor`
- [ ] Verify `make verify` passes

**File**: go.mod

---

### Task 5.2: Update CAPI vendor dependencies
- [ ] Run `make update` to regenerate vendor and clients
- [ ] Verify CAPA v1beta2 types with SpotMarketOptions are available

---

### Task 5.3: Add unit tests for NodePool controller
- [ ] Add TestIsSpotEnabled() to aws_test.go
- [ ] Add TestAWSMachineTemplateWithSpot() to aws_test.go
- [ ] Add TestAWSTagsWithTerminationHandler() to aws_test.go
- [ ] Run `make test` and verify all tests pass

**File**: hypershift-operator/controllers/nodepool/aws_test.go

---

### Task 5.4: Add unit tests for CPO v2 component
- [ ] Create component_test.go
- [ ] Add TestPredicate()
- [ ] Add TestDeploymentAdapter()
- [ ] Add TestMachineHealthCheckAdapter()
- [ ] Run `make test` and verify all tests pass

**File**: control-plane-operator/controllers/hostedcontrolplane/v2/awsnodeterminationhandler/component_test.go

---

### Task 5.5: Run linting and verification
- [ ] Run `make lint-fix`
- [ ] Run `make verify`
- [ ] Run `make test`
- [ ] Fix any issues identified

---

### Task 5.6: Build and test locally
- [ ] Run `make build`
- [ ] Run `make hypershift-install-aws-dev`
- [ ] Run `make run-operator-locally-aws-dev`
- [ ] Manually test spot NodePool creation
- [ ] Verify spot instances are created successfully

---

### Task 5.7: Run E2E tests
- [ ] Run `make e2e TEST=TestSpotInstanceCreation`
- [ ] Run `make e2e TEST=TestSpotInterruptionHandling`
- [ ] Run `make e2e TEST=TestNodePool` (regression check)
- [ ] Verify all tests pass

---

## End-to-End Verification

### Create spot-enabled NodePool
- [ ] Add annotation `hypershift.openshift.io/enable-spot: "true"` to NodePool
- [ ] Create NodePool
- [ ] Validate AWSMachineTemplate has SpotMarketOptions
- [ ] Validate EC2 instances are spot instances
- [ ] Validate instances have `aws-node-termination-handler/managed` tag
- [ ] Validate nodes have `machine.openshift.io/interruptible-instance` label

### Validate termination handler deployment
- [ ] Check aws-node-termination-handler Deployment exists in HCP namespace
- [ ] Verify pod is running
- [ ] Verify logs show SQS queue monitoring
- [ ] Verify MachineHealthCheck exists with correct selector

### Simulate spot interruption
- [ ] Send interruption warning to SQS queue
- [ ] Watch node become cordoned
- [ ] Watch pods drain from node
- [ ] Terminate instance via EC2 API
- [ ] Watch MachineHealthCheck delete unhealthy Machine
- [ ] Watch new Machine/Node creation
- [ ] Verify NodePool maintains replica count

### Validate no regression
- [ ] Create regular (non-spot) NodePool
- [ ] Verify instances are on-demand
- [ ] Verify no spot-related tags/labels
- [ ] Verify termination handler ignores these instances

---

## Success Criteria

- [ ] NodePools with annotation `hypershift.openshift.io/enable-spot: "true"` create EC2 spot instances via CAPA
- [ ] Spot instances tagged with `aws-node-termination-handler/managed`
- [ ] Nodes labeled with `machine.openshift.io/interruptible-instance`
- [ ] aws-node-termination-handler Deployment running in HCP namespace
- [ ] MachineHealthCheck created with correct selector
- [ ] E2E test validates spot instance creation
- [ ] E2E test validates interruption handling (drain → terminate → replace)
- [ ] No regressions in existing NodePool functionality
- [ ] All unit tests pass
- [ ] `make lint-fix` and `make verify` pass

---

## Configuration Required

### HostedCluster
Add annotation: `hypershift.openshift.io/aws-termination-handler-queue-url: "<queue-url>"`

### NodePool
Add annotation: `hypershift.openshift.io/enable-spot: "true"`

---

## Notes

- Image: `public.ecr.aws/aws-ec2/aws-node-termination-handler:v1.25.3`
- MachineHealthCheck timeout: 300s (5 minutes)
- Features enabled: SQS draining, spot interruption draining
- Features disabled: Rebalance draining (for initial PR)
