# HyperShift GCP Platform Support

This document provides comprehensive information about the GCP platform support in HyperShift, including cluster creation, infrastructure management, cost optimization, security best practices, and monitoring.

## Overview

The GCP platform implementation for HyperShift provides:

- **Cluster Management**: Create and destroy hosted clusters on Google Cloud Platform
- **Infrastructure Provisioning**: Automated creation of VPC, subnets, firewall rules, and Cloud NAT
- **IAM Management**: Service account creation and role assignment
- **NodePool Support**: GCP-specific worker node configurations
- **Cost Optimization**: Preemptible instances, sustained use discounts, and cost analysis
- **Security Best Practices**: Workload Identity, KMS encryption, network security
- **Monitoring & Observability**: Cloud Logging, Cloud Monitoring, Error Reporting

## Prerequisites

1. **GCP Project**: A valid GCP project with billing enabled
2. **Service Account**: A service account with appropriate permissions
3. **APIs Enabled**: Required GCP APIs must be enabled
4. **Credentials**: Service account key file in JSON format

### Required GCP APIs

```bash
# Enable required APIs
gcloud services enable compute.googleapis.com
gcloud services enable container.googleapis.com
gcloud services enable dns.googleapis.com
gcloud services enable iam.googleapis.com
gcloud services enable cloudkms.googleapis.com
gcloud services enable logging.googleapis.com
gcloud services enable monitoring.googleapis.com
gcloud services enable secretmanager.googleapis.com
```

### Required IAM Permissions

Your service account needs the following roles:

- `roles/compute.admin`
- `roles/storage.admin`
- `roles/dns.admin`
- `roles/iam.serviceAccountUser`
- `roles/cloudsql.admin`
- `roles/logging.admin`
- `roles/monitoring.admin`
- `roles/cloudkms.admin`
- `roles/secretmanager.admin`

## Quick Start

### 1. Create GCP Infrastructure

```bash
# Create infrastructure resources
hypershift create infra gcp \
  --project my-gcp-project \
  --region us-central1 \
  --zone us-central1-a \
  --infra-id my-cluster \
  --name my-cluster \
  --gcp-creds /path/to/service-account.json \
  --base-domain example.com
```

### 2. Create IAM Resources

```bash
# Create IAM resources
hypershift create infra gcp iam \
  --project my-gcp-project \
  --infra-id my-cluster \
  --gcp-creds /path/to/service-account.json
```

### 3. Create Hosted Cluster

```bash
# Create the hosted cluster
hypershift create cluster gcp \
  --name my-cluster \
  --namespace clusters \
  --project my-gcp-project \
  --region us-central1 \
  --zone us-central1-a \
  --instance-type n1-standard-4 \
  --disk-type pd-standard \
  --disk-size 100 \
  --gcp-creds /path/to/service-account.json \
  --pull-secret /path/to/pull-secret.json \
  --ssh-key /path/to/ssh-key.pub
```

### 4. Create NodePool

```bash
# Create a worker node pool
hypershift create nodepool gcp \
  --cluster-name my-cluster \
  --name workers \
  --namespace clusters \
  --replicas 3 \
  --gcp-instance-type n1-standard-4 \
  --gcp-disk-type pd-standard \
  --gcp-disk-size 100
```

## Configuration Options

### Cluster Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `--project` | GCP project ID | Required |
| `--region` | GCP region | us-central1 |
| `--zone` | GCP zone | us-central1-a |
| `--instance-type` | Machine type for nodes | n1-standard-4 |
| `--disk-type` | Disk type (pd-standard, pd-ssd, pd-balanced) | pd-standard |
| `--disk-size` | Disk size in GB (20-65536) | 100 |
| `--network` | Existing VPC network name | Auto-created |
| `--subnetwork` | Existing subnet name | Auto-created |
| `--preemptible` | Use preemptible instances | false |
| `--on-host-maintenance` | Maintenance behavior (MIGRATE/TERMINATE) | MIGRATE |

### Security Options

| Option | Description | Default |
|--------|-------------|---------|
| `--kms-key-name` | Cloud KMS key for etcd encryption | None |
| `--service-account-email` | Service account for instances | Auto-created |
| `--labels` | Additional labels for resources | None |
| `--resource-tags` | Additional tags for resources | None |

### Network Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `--control-plane-subnet` | Subnet for control plane | Same as main subnet |
| `--load-balancer-subnet` | Subnet for load balancers | Same as main subnet |

## Cost Optimization

### Preemptible Instances

Use preemptible instances for significant cost savings (up to 80%):

```bash
hypershift create cluster gcp \
  --name my-cluster \
  --preemptible \
  # ... other options
```

### Sustained Use Discounts

Sustained use discounts are automatically applied for VMs running >25% of the month.

### Committed Use Discounts

For predictable workloads, consider committed use discounts:
- 1-year commitment: up to 57% savings
- 3-year commitment: up to 70% savings

### Cost Analysis

```bash
# Analyze cost optimization opportunities
hypershift infra gcp analyze-costs \
  --project my-gcp-project \
  --infra-id my-cluster \
  --gcp-creds /path/to/service-account.json
```

## Security Best Practices

### Workload Identity

Enable Workload Identity for secure access to GCP services:

```bash
hypershift create cluster gcp \
  --name my-cluster \
  --enable-workload-identity \
  # ... other options
```

### KMS Encryption

Use Cloud KMS for etcd encryption:

```bash
hypershift create cluster gcp \
  --name my-cluster \
  --kms-key-name projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key \
  # ... other options
```

### Network Security

The implementation automatically creates secure firewall rules:
- Deny all ingress by default
- Allow only necessary internal communication
- Restrict SSH access
- Enable VPC Flow Logs (optional)

### Security Analysis

```bash
# Apply security best practices
hypershift infra gcp apply-security \
  --project my-gcp-project \
  --infra-id my-cluster \
  --enable-workload-identity \
  --enable-kms-encryption \
  --enable-vpc-flow-logs \
  --gcp-creds /path/to/service-account.json
```

## Monitoring and Observability

### Cloud Logging

Automatically configured for:
- Kubernetes cluster logs
- Node logs
- Pod logs
- Application logs

### Cloud Monitoring

Includes pre-configured:
- Node CPU and memory metrics
- Pod metrics
- Cluster health alerts
- Custom dashboards

### Setup Monitoring

```bash
# Setup comprehensive monitoring
hypershift infra gcp setup-monitoring \
  --project my-gcp-project \
  --infra-id my-cluster \
  --enable-cloud-logging \
  --enable-cloud-monitoring \
  --enable-error-reporting \
  --enable-cloud-trace \
  --gcp-creds /path/to/service-account.json
```

## Maintenance and Operations

### Cluster Scaling

```bash
# Scale nodepool
hypershift scale nodepool workers \
  --namespace clusters \
  --replicas 5
```

### Cluster Upgrades

```bash
# Upgrade cluster
hypershift upgrade cluster \
  --name my-cluster \
  --namespace clusters \
  --release-image quay.io/openshift-release-dev/ocp-release:4.14.0-x86_64
```

### Backup and Recovery

Regular backups are recommended:
- etcd snapshots
- Persistent volume snapshots
- Configuration backups

### Cleanup

```bash
# Destroy cluster and infrastructure
hypershift destroy cluster gcp \
  --name my-cluster \
  --namespace clusters \
  --project my-gcp-project \
  --region us-central1 \
  --gcp-creds /path/to/service-account.json
```

## Troubleshooting

### Common Issues

1. **Authentication Errors**
   - Verify service account permissions
   - Check API enablement
   - Validate credentials file

2. **Network Issues**
   - Check firewall rules
   - Verify subnet configuration
   - Ensure Cloud NAT is configured

3. **Resource Limits**
   - Check GCP quotas
   - Verify region availability
   - Monitor resource usage

### Debug Commands

```bash
# Check cluster status
hypershift get cluster my-cluster -n clusters

# Check nodepool status
hypershift get nodepool workers -n clusters

# View cluster logs
hypershift logs cluster my-cluster -n clusters
```

## Best Practices

1. **Resource Organization**
   - Use consistent naming conventions
   - Apply appropriate labels and tags
   - Organize resources by environment

2. **Security**
   - Enable Workload Identity
   - Use KMS encryption
   - Implement network policies
   - Regular security audits

3. **Cost Management**
   - Use preemptible instances where appropriate
   - Monitor resource utilization
   - Implement autoscaling
   - Regular cost analysis

4. **Monitoring**
   - Set up comprehensive monitoring
   - Configure alerting
   - Regular health checks
   - Performance optimization

5. **Backup and Recovery**
   - Regular backups
   - Test recovery procedures
   - Document disaster recovery plans

## Limitations

1. **Regional Restrictions**
   - Some GCP services may not be available in all regions
   - Consider data residency requirements

2. **Quota Limits**
   - GCP has default quotas that may need adjustment
   - Plan for quota increases in advance

3. **Network Limitations**
   - Private clusters require additional configuration
   - Consider network topology carefully

## Support and Documentation

- [GCP Documentation](https://cloud.google.com/docs)
- [HyperShift Documentation](https://hypershift-docs.netlify.app/)
- [OpenShift Documentation](https://docs.openshift.com/)

For issues and feature requests, please use the HyperShift GitHub repository.