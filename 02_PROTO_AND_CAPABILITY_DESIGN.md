# Proto Specification & Capability Negotiation Design

**Goal:** Separate proto spec + implement capability-based feature negotiation  
**Timeline:** 7 weeks (4 weeks proto + 3 weeks capability)

---

## Part 1: Proto Specification Separation

**Inspired by:** https://github.com/csi-addons/spec  
**Pattern:** Standalone specification repository with language bindings

### Current State

Provider API **already has separate `go.mod`**:
```
ocs-operator/services/provider/api/v4/
├── go.mod  ← SEPARATE MODULE!
├── provider.pb.go
├── provider_grpc.pb.go
├── interfaces/interfaces.go
└── client/client.go
```

**Module:** `github.com/red-hat-storage/ocs-operator/services/provider/api/v4`

**Key Insight:** 90% done! Just needs extraction to standalone repo.

---

### Proposed Structure: `odf-provider-spec`

```
odf-provider-spec/
├── README.md
├── LICENSE
├── SPEC.md                         # Human-readable API spec
├── COMPATIBILITY.md                # Version compatibility matrix
├── CHANGELOG.md                    # API changes per version
├── spec/
│   ├── v1/
│   │   ├── provider.proto          # v1 proto definition
│   │   └── README.md               # v1 notes
│   └── v2/                         # Future major version
│       └── provider.proto
├── lib/
│   ├── go/                         # Go bindings
│   │   ├── go.mod
│   │   ├── provider/v1/
│   │   │   ├── provider.pb.go
│   │   │   ├── provider_grpc.pb.go
│   │   │   ├── client.go
│   │   │   └── interfaces.go
│   │   └── provider/v2/
│   ├── python/                     # Future: Python bindings
│   └── java/                       # Future: Java bindings
├── docs/
│   ├── migration/
│   │   └── v1-to-v2.md
│   ├── evolution-policy.md         # Proto evolution rules
│   └── examples/
│       ├── onboarding.md
│       └── capability-negotiation.md
└── .github/workflows/
    ├── generate-proto.yml          # Auto-generate bindings
    └── compatibility-test.yml      # Test backward compat
```

---

### Implementation Steps

#### Week 1: Repository Setup

1. **Create repository**
   ```bash
   gh repo create red-hat-storage/odf-provider-spec --public
   cd odf-provider-spec
   mkdir -p spec/v1 lib/go/provider/v1 docs/examples .github/workflows
   ```

2. **Copy files from ocs-operator**
   ```bash
   # From ocs-operator
   cp services/provider/proto/provider.proto spec/v1/
   cp services/provider/api/provider.pb.go lib/go/provider/v1/
   cp services/provider/api/provider_grpc.pb.go lib/go/provider/v1/
   cp services/provider/api/interfaces/interfaces.go lib/go/provider/v1/
   cp services/provider/api/client/client.go lib/go/provider/v1/
   cp services/provider/api/storageclient.go lib/go/provider/v1/
   ```

3. **Create go.mod**
   ```bash
   cd lib/go
   go mod init github.com/red-hat-storage/odf-provider-spec/lib/go
   go mod tidy
   ```

4. **Tag v1.0.0**
   ```bash
   git add .
   git commit -m "Initial import from ocs-operator"
   git tag v1.0.0
   git push --tags
   ```

#### Week 2: Documentation

Create comprehensive documentation:
- **SPEC.md:** Human-readable API documentation
- **COMPATIBILITY.md:** Version matrix
- **docs/evolution-policy.md:** Rules for proto changes

#### Week 3: Migrate ocs-operator

1. Update go.mod:
   ```go
   require (
       github.com/red-hat-storage/odf-provider-spec/lib/go v1.0.0
   )
   ```

2. Update imports:
   ```go
   // Old
   import pb "github.com/red-hat-storage/ocs-operator/services/provider/api/v4"
   
   // New
   import pb "github.com/red-hat-storage/odf-provider-spec/lib/go/provider/v1"
   ```

3. Remove local `services/provider/api/`

#### Week 4: Migrate ocs-client-operator

Same process as ocs-operator.

---

### Proto Evolution Policy

#### Rules (Must Follow)

✅ **ALLOWED (Minor Version Bump):**
- Add new optional field
- Add new RPC
- Add new enum value

❌ **NOT ALLOWED (Requires Major Version):**
- Remove field (use `reserved` instead)
- Remove RPC (mark deprecated instead)
- Change field type
- Change enum value

#### Example: Adding New Feature

```protobuf
// v1.1.0 - MINOR bump (backward compatible)
message OnboardConsumerRequest {
    string onboardingTicket = 1;
    string consumerName = 2;
    string clientOperatorVersion = 3;
    
    // NEW in v1.1.0 - optional field
    repeated Capability capabilities = 20;
}
```

---

## Part 2: Capability Negotiation

**Pattern:** Client advertises capabilities using CSI AddOns-style structure  
**Storage:** `StorageConsumer.Status.Capabilities`  
**Definition:** Proto package

### Proto Design

```protobuf
// services/provider/proto/provider.proto

// Capability represents a feature that a client supports
// Structured using oneof for type-safe categorization
message Capability {
    oneof type {
        Service service = 1;
        Class class = 2;
        Feature feature = 3;
    }
    
    // Service capability - Storage services
    message Service {
        Type type = 1;
        
        enum Type {
            UNKNOWN = 0;
            RBD = 1;           // RBD block storage
            CEPHFS = 2;        // CephFS filesystem
            NVMEOF = 3;        // NVMe-oF storage
            NFS = 4;           // NFS storage
            NOOBAA_OBC = 5;    // NooBaa Object Bucket Claims
            RGW = 6;           // RADOS Gateway
        }
    }
    
    // Class capability - K8s resource classes
    message Class {
        Type type = 1;
        
        enum Type {
            UNKNOWN = 0;
            STORAGE_CLASS = 1;
            VOLUME_SNAPSHOT_CLASS = 2;
            VOLUME_GROUP_SNAPSHOT_CLASS = 3;
            VOLUME_REPLICATION_CLASS = 4;
            VOLUME_GROUP_REPLICATION_CLASS = 5;
            NETWORK_FENCE_CLASS = 6;
            VOLUME_ATTRIBUTES_CLASS = 7;
        }
    }
    
    // Feature capability - Client features
    message Feature {
        Type type = 1;
        
        enum Type {
            UNKNOWN = 0;
            ENHANCED_AUTH = 1;               // Enhanced authentication/authorization
            NOTIFY = 2;                      // Notify RPC for OBC events
            CLIENT_ALERTS = 3;               // GetClientAlerts RPC
            CEPHFS_METRICS = 4;              // CephFS metrics reporting
            CLIENT_PROFILE = 5;              // CSI ClientProfile support
            CLIENT_PROFILE_MAPPING = 6;      // ClientProfileMapping
            CLIENT_PROFILE_REPLICATION = 7;  // ClientProfileReplication
            CEPH_CONNECTION = 8;             // CephConnection resource
        }
    }
}
```

### OnboardConsumer Changes

```protobuf
message OnboardConsumerRequest {
    // Existing fields (v1.0)
    string onboardingTicket = 1;
    string consumerName = 2;
    string clientOperatorVersion = 3;
    string clientPlatformVersion = 4;
    string clientOperatorNamespace = 5;
    string clientID = 6;
    string clientName = 7;
    string clusterID = 8;
    string clusterName = 9;
    
    // NEW in v1.1 - OPTIONAL
    // Client advertises its capabilities
    // If empty, server infers from clientOperatorVersion
    repeated Capability capabilities = 20;
}

message OnboardConsumerResponse {
    string storageConsumerUUID = 1;
    
    // NEW in v1.1 - OPTIONAL
    ServerInfo serverInfo = 2;
}

message ServerInfo {
    string version = 1;
    repeated Capability serverCapabilities = 2;
    string minSupportedClientVersion = 3;
    string maxSupportedClientVersion = 4;
    repeated Capability enabledCapabilities = 5;  // Intersection
}
```

---

### Server Implementation

#### Capability Inference (for old clients)

```go
// services/provider/server/capabilities.go

func inferCapabilitiesFromVersion(version semver.Version) []*pb.Capability {
    var caps []*pb.Capability
    
    // Base capabilities (all versions)
    caps = append(caps,
        &pb.Capability{Type: &pb.Capability_Service{
            Service: &pb.Capability_Service{Type: pb.Capability_Service_RBD},
        }},
        &pb.Capability{Type: &pb.Capability_Service{
            Service: &pb.Capability_Service{Type: pb.Capability_Service_CEPHFS},
        }},
    )
    
    // v5.1+ capabilities
    if version.GTE(semver.MustParse("5.1.0")) {
        caps = append(caps,
            &pb.Capability{Type: &pb.Capability_Service{
                Service: &pb.Capability_Service{Type: pb.Capability_Service_NVMEOF},
            }},
            &pb.Capability{Type: &pb.Capability_Class{
                Class: &pb.Capability_Class{Type: pb.Capability_Class_VOLUME_ATTRIBUTES_CLASS},
            }},
        )
    }
    
    return caps
}
```

#### OnboardConsumer Handler

```go
func (s *OCSProviderServer) OnboardConsumer(ctx context.Context, req *pb.OnboardConsumerRequest) (*pb.OnboardConsumerResponse, error) {
    logger := klog.FromContext(ctx).WithName("OnboardConsumer")
    
    clientVersion, err := semver.Parse(req.ClientOperatorVersion)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid version: %v", err)
    }
    
    // Check version compatibility (configurable - see Document 3)
    if err := s.checkVersionCompatibility(clientVersion); err != nil {
        return nil, err
    }
    
    // Determine capabilities (hybrid approach)
    var clientCapabilities []*pb.Capability
    if len(req.Capabilities) > 0 {
        // Client sent capabilities (v5.1+ or backported)
        clientCapabilities = req.Capabilities
        logger.Info("Client advertised capabilities", "count", len(clientCapabilities))
    } else {
        // Old client - infer from version
        clientCapabilities = inferCapabilitiesFromVersion(clientVersion)
        logger.Info("Inferred capabilities from version", "count", len(clientCapabilities))
    }
    
    // Create StorageConsumer
    consumer := &ocsv1alpha1.StorageConsumer{ /* ... */ }
    if err := s.consumerManager.Create(ctx, consumer, req); err != nil {
        return nil, status.Errorf(codes.Internal, "failed to create consumer: %v", err)
    }
    
    // Store capabilities in Status
    consumer.Status.Capabilities = clientCapabilities
    if err := s.consumerManager.UpdateStatus(ctx, consumer); err != nil {
        logger.Error(err, "Failed to update consumer status")
    }
    
    // Build server info
    serverInfo := s.buildServerInfo(clientCapabilities)
    
    return &pb.OnboardConsumerResponse{
        StorageConsumerUUID: string(consumer.UID),
        ServerInfo:          serverInfo,
    }, nil
}
```

#### GetDesiredClientState Using Capabilities

```go
func (s *OCSProviderServer) GetDesiredClientState(ctx context.Context, req *pb.GetDesiredClientStateRequest) (*pb.GetDesiredClientStateResponse, error) {
    consumer, err := s.consumerManager.Get(ctx, req.StorageConsumerUUID)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "consumer not found")
    }
    
    // Get capabilities from Status
    capabilities := consumer.Status.Capabilities
    
    response := &pb.GetDesiredClientStateResponse{
        ClientOperatorChannel: channelName,
        MaintenanceMode:       inMaintenanceMode,
        MirrorEnabled:         isConsumerMirrorEnabled,
    }
    
    // Build resources based on capabilities
    response.KubeObjects = s.buildKubeObjects(consumer, capabilities)
    
    // Conditionally add driver requirements
    if hasCapability(capabilities, pb.Capability_Service_NVMEOF) {
        response.NvmeofDriverRequirements = s.buildNvmeofRequirements(consumer)
    }
    
    return response, nil
}
```

---

### Client Implementation

```go
// ocs-client-operator/internal/controller/storageclient_controller.go

func (r *storageClientReconcile) onboardConsumer(client *providerClient.OCSProviderClient, operatorVersion string) error {
    req := providerClient.NewOnboardConsumerRequest()
    req.SetOnboardingTicket(r.storageClient.Spec.OnboardingTicket)
    req.SetClientOperatorVersion(operatorVersion)
    // ... other fields ...
    
    // Detect and advertise capabilities (v5.1+ clients)
    clientVersion, _ := semver.Parse(operatorVersion)
    if clientVersion.GTE(semver.MustParse("5.1.0")) {
        capabilities := r.detectCapabilities()
        req.SetCapabilities(capabilities)
    }
    
    response, err := client.OnboardConsumer(r.ctx, req)
    if err != nil {
        return fmt.Errorf("failed to onboard: %v", err)
    }
    
    // Log enabled capabilities
    if response.ServerInfo != nil {
        for _, cap := range response.ServerInfo.EnabledCapabilities {
            r.logCapability(cap)
        }
    }
    
    return nil
}

func (r *storageClientReconcile) detectCapabilities() []*pb.Capability {
    var caps []*pb.Capability
    
    // Base services
    caps = append(caps,
        &pb.Capability{Type: &pb.Capability_Service{
            Service: &pb.Capability_Service{Type: pb.Capability_Service_RBD},
        }},
    )
    
    // Conditional capabilities
    if r.hasNVMeOFSupport() {
        caps = append(caps, &pb.Capability{Type: &pb.Capability_Service{
            Service: &pb.Capability_Service{Type: pb.Capability_Service_NVMEOF},
        }})
    }
    
    // Backported features
    if r.hasEnhancedAuth() {
        caps = append(caps, &pb.Capability{Type: &pb.Capability_Feature{
            Feature: &pb.Capability_Feature{Type: pb.Capability_Feature_ENHANCED_AUTH},
        }})
    }
    
    return caps
}
```

---

### StorageConsumer API

```go
// api/v1alpha1/storageconsumer_types.go

type StorageConsumerStatus struct {
    // ... existing fields ...
    
    // Capabilities advertised by client, stored by server
    // +optional
    Capabilities []*Capability `json:"capabilities,omitempty"`
}

// Capability is stored in Status (proto message directly)
```

---

### Example: Backported Feature

**Scenario:** Enhanced Auth backported to v4.22.5

**Client v4.22.5:**
```go
caps := []*pb.Capability{
    // Base capabilities...
    &pb.Capability{Type: &pb.Capability_Feature{
        Feature: &pb.Capability_Feature{Type: pb.Capability_Feature_ENHANCED_AUTH},
    }},  // Backported!
}
```

**Server:**
- Stores in `Status.Capabilities`
- Sends enhanced auth resources
- Works even though version is v4.22!

---

## Benefits

### Proto Separation

✅ **Language-agnostic** - Can generate Python, Java bindings  
✅ **Independent versioning** - API evolves separately from operators  
✅ **Clear compatibility** - COMPATIBILITY.md shows support matrix  
✅ **Third-party ready** - Others can implement provider API  

### Capability Negotiation

✅ **Handles backports** - Client declares what it actually has  
✅ **Type-safe** - Proto generates type-safe code  
✅ **Backward compatible** - Old clients work via version inference  
✅ **Extensible** - Easy to add new capabilities  
✅ **Observable** - Capabilities stored in Status  

---

## Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| Proto repo setup | 1 week | odf-provider-spec v1.0.0 |
| Documentation | 1 week | SPEC.md, COMPATIBILITY.md |
| Migrate ocs-operator | 1 week | Using odf-provider-spec |
| Migrate ocs-client-operator | 1 week | Using odf-provider-spec |
| **Proto separation done** | **4 weeks** | |
| Capability proto design | 1 week | Capability message |
| Server implementation | 1 week | Inference + storage |
| Client implementation | 1 week | Detection + advertisement |
| **Capability done** | **3 weeks** | |
| **Total** | **7 weeks** | |

---

## Next Steps

1. Get approval for `odf-provider-spec` repository creation
2. Start proto separation (Week 1)
3. Parallel: Design capability negotiation
4. Test with multiple client versions

**See Document 3 for version configuration and upgrade safety.**
