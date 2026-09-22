# Version Configuration & Provider Upgrade Safety

**Goal:** Configurable version policy + prevent provider upgrades that break clients  
**Timeline:** 4 weeks (1 week configuration + 3 weeks upgrade safety)

---

## Part 1: Configurable Version Check

**Problem:** Hard-coded version checks lack operational flexibility  
**Solution:** Environment variable-based configuration

### Environment Variables

```bash
# Option 1: Explicit minimum (recommended)
MIN_SUPPORTED_CLIENT_VERSION="4.20.0"

# Option 2: Skew-based (flexible)
SUPPORTED_CLIENT_MINOR_SKEW="4"  # n-4

# Optional: Maximum (defaults to provider version)
MAX_SUPPORTED_CLIENT_VERSION=""

# Emergency: Disable check (UNSAFE!)
DISABLE_VERSION_CHECK="false"
```

### Configuration Priority

1. `DISABLE_VERSION_CHECK=true` → All checks disabled (dangerous!)
2. `MIN_SUPPORTED_CLIENT_VERSION` set → Use explicit minimum
3. `SUPPORTED_CLIENT_MINOR_SKEW` set → Compute from skew
4. **Default** → n-4 from provider version

---

### Implementation

```go
// services/provider/server/config.go

type VersionCheckConfig struct {
    MinSupportedClientVersion semver.Version
    MaxSupportedClientVersion semver.Version
    DisableVersionCheck       bool
}

func LoadVersionCheckConfig(providerVersion semver.Version) (*VersionCheckConfig, error) {
    cfg := &VersionCheckConfig{}
    
    // Check if disabled
    if os.Getenv("DISABLE_VERSION_CHECK") == "true" {
        cfg.DisableVersionCheck = true
        return cfg, nil
    }
    
    // Try explicit minimum first
    if minVerStr := os.Getenv("MIN_SUPPORTED_CLIENT_VERSION"); minVerStr != "" {
        cfg.MinSupportedClientVersion = semver.MustParse(minVerStr)
    } else {
        // Fall back to skew-based
        skewStr := os.Getenv("SUPPORTED_CLIENT_MINOR_SKEW")
        if skewStr == "" {
            skewStr = "4" // Default: n-4
        }
        skew, _ := strconv.Atoi(skewStr)
        cfg.MinSupportedClientVersion = computeMinVersionFromSkew(providerVersion, skew)
    }
    
    // Maximum version
    if maxVerStr := os.Getenv("MAX_SUPPORTED_CLIENT_VERSION"); maxVerStr != "" {
        cfg.MaxSupportedClientVersion = semver.MustParse(maxVerStr)
    } else {
        cfg.MaxSupportedClientVersion = providerVersion
    }
    
    return cfg, nil
}

func computeMinVersionFromSkew(providerVersion semver.Version, skew int) semver.Version {
    minMinor := int(providerVersion.Minor) - skew
    if minMinor < 0 {
        minMinor = 0
    }
    return semver.Version{Major: providerVersion.Major, Minor: uint64(minMinor), Patch: 0}
}
```

### OnboardConsumer with Configurable Check

```go
type OCSProviderServer struct {
    versionCheckConfig *VersionCheckConfig
}

func (s *OCSProviderServer) OnboardConsumer(ctx context.Context, req *pb.OnboardConsumerRequest) (*pb.OnboardConsumerResponse, error) {
    clientVersion, err := semver.Parse(req.ClientOperatorVersion)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid version: %v", err)
    }
    
    // Check version compatibility (configurable!)
    if err := s.checkVersionCompatibility(clientVersion); err != nil {
        return nil, err
    }
    
    // ... rest of onboarding
}

func (s *OCSProviderServer) checkVersionCompatibility(clientVersion semver.Version) error {
    cfg := s.versionCheckConfig
    
    if cfg.DisableVersionCheck {
        return nil // Dangerous but configurable
    }
    
    // Check minimum
    if clientVersion.LT(cfg.MinSupportedClientVersion) {
        return status.Errorf(codes.FailedPrecondition,
            "client version %s not supported, minimum is %s",
            clientVersion, cfg.MinSupportedClientVersion)
    }
    
    // Check maximum
    if clientVersion.GT(cfg.MaxSupportedClientVersion) {
        return status.Errorf(codes.FailedPrecondition,
            "client version %s ahead of server, maximum is %s",
            clientVersion, cfg.MaxSupportedClientVersion)
    }
    
    return nil
}
```

---

### CSV Configuration

```yaml
# deploy/csv-templates/ocs-operator.csv.yaml.in

spec:
  install:
    spec:
      deployments:
      - name: ocs-operator
        spec:
          template:
            spec:
              containers:
              - name: ocs-operator
                env:
                # Version check configuration
                - name: MIN_SUPPORTED_CLIENT_VERSION
                  value: "4.20.0"
                
                - name: SUPPORTED_CLIENT_MINOR_SKEW
                  value: "4"
                
                - name: DISABLE_VERSION_CHECK
                  value: "false"
```

### Override via Subscription

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ocs-operator
spec:
  package: ocs-operator
  channel: stable-5.1
  config:
    env:
    # Override: Use n-2 instead of n-4
    - name: SUPPORTED_CLIENT_MINOR_SKEW
      value: "2"
```

---

### Configuration Examples

| Scenario | Config | Result (v5.1) |
|----------|--------|---------------|
| **Default n-4** | `SKEW="4"` | v4.20.0 - v5.1.x |
| **Explicit range** | `MIN="4.22.0"` | v4.22.0 - v5.1.x |
| **Conservative n-2** | `SKEW="2"` | v4.23.0 - v5.1.x |
| **Same version** | `SKEW="0"` | v5.1.0 - v5.1.x |
| **Testing only** | `DISABLE="true"` | ANY (unsafe!) |

---

## Part 2: Provider Upgrade Safety

**Problem:** Provider upgrades can silently break connected clients  
**Solution:** Pre-upgrade validation + admission control + alerts

### The Problem

```
Provider v5.1 → v5.2 upgrade
Has clients: v4.20, v4.21, v5.0
v5.2 minimum: v4.22 (n-4)
❌ Clients v4.20, v4.21 break silently!
```

**Impact:** Production outage, storage provisioning fails, no warning

---

### Solution Components

#### 1. Compatibility Matrix in Operator

```go
// pkg/util/compatibility/matrix.go

var DefaultMatrix = map[string]string{
    "5.0": "4.16.0",
    "5.1": "4.20.0",
    "5.2": "4.22.0",
    "5.3": "4.23.0",
    "6.0": "5.0.0",
}

func GetMinimumClientVersion(providerVersion string) (string, error) {
    pv := semver.MustParse(providerVersion)
    key := fmt.Sprintf("%d.%d", pv.Major, pv.Minor)
    
    if minClient, ok := DefaultMatrix[key]; ok {
        return minClient, nil
    }
    
    return "", fmt.Errorf("no compatibility info for version %s", providerVersion)
}

func GetIncompatibleConsumers(consumers []ConsumerVersion, targetVersion string) ([]ConsumerVersion, error) {
    minClient, err := GetMinimumClientVersion(targetVersion)
    if err != nil {
        return nil, err
    }
    
    minVer := semver.MustParse(minClient)
    var incompatible []ConsumerVersion
    
    for _, consumer := range consumers {
        cv := semver.MustParse(consumer.Version)
        if cv.LT(minVer) {
            incompatible = append(incompatible, consumer)
        }
    }
    
    return incompatible, nil
}
```

#### 2. Pre-Upgrade Validation (Init Container)

```bash
#!/bin/bash
# validate-upgrade.sh - runs as init container before OLM upgrades

TARGET_VERSION="${TARGET_VERSION:-5.2.0}"
NAMESPACE="${WATCH_NAMESPACE:-openshift-storage}"

# Get minimum client version for target
case "${TARGET_VERSION}" in
  5.1.*) MIN_CLIENT="4.20.0" ;;
  5.2.*) MIN_CLIENT="4.22.0" ;;
  *)     echo "Unknown version"; exit 1 ;;
esac

echo "Target ${TARGET_VERSION} requires minimum client ${MIN_CLIENT}"

# Check all StorageConsumers
INCOMPATIBLE=0
for consumer in $(kubectl get storageconsumers -n ${NAMESPACE} -o name); do
  CLIENT_VERSION=$(kubectl get ${consumer} -n ${NAMESPACE} \
    -o jsonpath='{.status.client.operatorVersion}')
  
  if [ -z "${CLIENT_VERSION}" ]; then
    echo "WARNING: ${consumer} has no version info"
    continue
  fi
  
  # Compare versions
  if ! semver-compare "${CLIENT_VERSION}" ">=" "${MIN_CLIENT}"; then
    echo "ERROR: ${consumer} version ${CLIENT_VERSION} < ${MIN_CLIENT}"
    INCOMPATIBLE=$((INCOMPATIBLE + 1))
  fi
done

if [ ${INCOMPATIBLE} -gt 0 ]; then
  echo ""
  echo "=========================================="
  echo "UPGRADE BLOCKED"
  echo "=========================================="
  echo "${INCOMPATIBLE} client(s) incompatible with ${TARGET_VERSION}"
  echo "Minimum required: ${MIN_CLIENT}"
  echo ""
  echo "Action: Upgrade incompatible clients first"
  exit 1
fi

echo "All clients compatible. Upgrade can proceed."
exit 0
```

**CSV Integration:**
```yaml
spec:
  install:
    spec:
      deployments:
      - name: ocs-operator
        spec:
          template:
            spec:
              initContainers:
              - name: pre-upgrade-validation
                image: ocs-operator:5.2.0
                command: ["/usr/local/bin/validate-upgrade"]
                env:
                - name: TARGET_VERSION
                  value: "5.2.0"
```

#### 3. Admission Webhook

```go
// internal/webhook/subscription_validator.go

type SubscriptionValidator struct {
    Client client.Client
}

var compatibilityMatrix = map[string]string{
    "stable-5.1": "4.20.0",
    "stable-5.2": "4.22.0",
}

func (v *SubscriptionValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
    subscription := &opv1a1.Subscription{}
    if err := v.Decoder.Decode(req, subscription); err != nil {
        return admission.Errored(http.StatusBadRequest, err)
    }
    
    // Only validate ocs-operator
    if subscription.Spec.Package != "ocs-operator" {
        return admission.Allowed("not ocs-operator")
    }
    
    targetChannel := subscription.Spec.Channel
    minClientVersion := compatibilityMatrix[targetChannel]
    
    // Check all connected clients
    consumers := &ocsv1alpha1.StorageConsumerList{}
    if err := v.Client.List(ctx, consumers); err != nil {
        return admission.Errored(http.StatusInternalServerError, err)
    }
    
    var incompatible []string
    for _, consumer := range consumers.Items {
        if consumer.Status.Client == nil {
            continue
        }
        
        clientVersion := consumer.Status.Client.OperatorVersion
        if !isVersionCompatible(clientVersion, minClientVersion) {
            incompatible = append(incompatible, 
                fmt.Sprintf("%s (v%s)", consumer.Name, clientVersion))
        }
    }
    
    if len(incompatible) > 0 {
        return admission.Denied(fmt.Sprintf(
            "Cannot upgrade to %s: %d incompatible clients.\n"+
            "Minimum required: %s\n"+
            "Incompatible: %v",
            targetChannel, len(incompatible), minClientVersion, incompatible))
    }
    
    return admission.Allowed("all clients compatible")
}
```

#### 4. Prometheus Metrics

```go
// metrics/collectors/consumer_version.go

var (
    consumerVersionGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ocs_provider_consumer_version_info",
            Help: "Version info for connected storage consumers",
        },
        []string{"consumer_name", "version"},
    )
    
    consumerVersionSkew = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ocs_provider_consumer_version_skew",
            Help: "Versions behind provider",
        },
        []string{"consumer_name", "skew_level"},
    )
    
    versionCheckConfigInfo = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ocs_provider_version_check_config",
            Help: "Version check configuration",
        },
        []string{"min_version", "max_version", "enabled"},
    )
)
```

#### 5. Prometheus Alerts

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: ocs-consumer-version-alerts
spec:
  groups:
  - name: ocs.consumer.version
    rules:
    # Alert when clients at n-4 boundary
    - alert: OCSConsumerAtVersionBoundary
      expr: ocs_provider_consumer_version_skew{skew_level="n-4"} > 0
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Consumer {{ $labels.consumer_name }} at version boundary"
        description: "Consumer is n-4 versions behind. Will break if provider upgrades."
    
    # Alert when upgrade would break clients
    - alert: OCSProviderUpgradeWouldBreakClients
      expr: ocs_provider_incompatible_consumers_total > 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Provider upgrade would break {{ $value }} consumer(s)"
        description: "Upgrade clients first before upgrading provider."
```

---

### Safe Upgrade Procedure

#### Step 1: Check Client Versions

```bash
kubectl get storageconsumers -A \
  -o custom-columns=NAME:.metadata.name,VERSION:.status.client.operatorVersion
```

#### Step 2: Check Compatibility

```bash
# For upgrade to v5.2 (min client v4.22)
kubectl get storageconsumers -A \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.client.operatorVersion}{"\n"}{end}' | \
  awk '$2 < "4.22.0" {print "INCOMPATIBLE:", $1, $2}'
```

#### Step 3: Check Alerts

Check Prometheus for `OCSProviderUpgradeWouldBreakClients` alert.

#### Step 4: Upgrade Incompatible Clients

Upgrade clients v4.20, v4.21 to v4.22+

#### Step 5: Proceed with Provider Upgrade

OLM will run pre-upgrade validation. Upgrade blocked if validation fails.

### If Upgrade is Blocked

**Error Message:**
```
UPGRADE BLOCKED
2 client(s) incompatible with 5.2.0
Minimum required: 4.22.0

Incompatible clients:
- client-cluster-1 (v4.20.0)
- client-cluster-2 (v4.21.0)

Action: Upgrade incompatible clients first
```

**Resolution:**
1. Upgrade clients to compatible version
2. Or offboard incompatible clients
3. Retry provider upgrade

---

## Implementation Timeline

### Week 1: Configurable Version Check

**Tasks:**
- Implement `LoadVersionCheckConfig()`
- Update `checkVersionCompatibility()`
- Add validation at startup
- Unit tests

**Deliverable:** Configurable version policy

### Week 2: Metrics & Compatibility Matrix

**Tasks:**
- Add consumer version metrics
- Add version skew metrics
- Embed compatibility matrix
- Create PrometheusRules

**Deliverable:** Observability and alerts

### Week 3: Pre-Upgrade Validation

**Tasks:**
- Create validation script
- Add as init container in CSV
- Test upgrade blocking
- Documentation

**Deliverable:** Upgrades blocked if incompatible clients

### Week 4: Admission Webhook (Optional)

**Tasks:**
- Create webhook server
- Register ValidatingWebhookConfiguration
- Test subscription changes
- Documentation

**Deliverable:** Real-time upgrade blocking

---

## Testing

### Test Scenarios

| Provider | Clients | Action | Expected |
|----------|---------|--------|----------|
| v5.1 | v5.1, v5.0, v4.22 | Upgrade to v5.2 | ✅ Allowed |
| v5.1 | v5.1, v4.21, v4.20 | Upgrade to v5.2 | ❌ Blocked |
| v5.1 | v4.20 | Check alerts | ⚠️ Alert fires |

### Configuration Tests

```bash
# Test default n-4
SUPPORTED_CLIENT_MINOR_SKEW=4
# Provider v5.1 accepts v4.20+

# Test explicit minimum
MIN_SUPPORTED_CLIENT_VERSION="4.22.0"
# Provider v5.1 accepts v4.22+

# Test disable (testing only!)
DISABLE_VERSION_CHECK="true"
# Provider v5.1 accepts ANY
```

---

## Effort Estimate

| Component | Story Points | Priority |
|-----------|--------------|----------|
| Configurable version check | 3 | HIGH |
| Compatibility matrix | 2 | HIGH |
| Metrics & alerts | 3 | HIGH |
| Pre-upgrade validation | 5 | HIGH |
| Admission webhook | 8 | MEDIUM |
| Documentation | 2 | HIGH |
| **Total** | **23 SP** | |

**Timeline:** 4 weeks

---

## Benefits

### Configurable Version Check

✅ No code changes for version policy adjustments  
✅ Gradual rollout support  
✅ Emergency override capability  
✅ Observable via metrics  
✅ Safe defaults (n-4)  

### Upgrade Safety

✅ Prevents breaking production clients  
✅ Pre-upgrade validation blocks bad upgrades  
✅ Alerts warn before issues  
✅ Metrics show version distribution  
✅ Clear error messages guide operators  

---

## Recommendations

✅ **Implement Both**

**Priority:** HIGH - Critical for operational safety

**Order:**
1. Configurable version check (Week 1)
2. Metrics & matrix (Week 2)
3. Pre-upgrade validation (Week 3)
4. Admission webhook (Week 4 - optional)

This prevents production incidents from provider upgrades!

---

**See Document 1 for complete analysis and Document 2 for proto/capability design.**
