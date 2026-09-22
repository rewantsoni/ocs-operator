# ODF n-4/n-5 Version Compatibility - Analysis & Feasibility

**Date:** 2026-09-16  
**Analyzed:** ocs-operator (provider) + ocs-client-operator (client)  
**Goal:** Support n-4/n-5 version compatibility for v5.1 release

---

## Executive Summary

### ✅ Proto Separation: READY (4 weeks)
Provider API already has separate `go.mod` - just needs extraction to `odf-provider-spec` repository.

### ⚠️ n-4/n-5 Support: FEASIBLE WITH CONSTRAINTS (14 weeks)
- **Minimum supported:** v4.20 (cannot support v4.19 and below due to breaking changes)
- **Support window for v5.1:** v4.20 through v5.1
- **Only 1 critical blocker:** Hard version check at server.go:236

### 🔴 Operational Gap: Provider Upgrade Safety (3 weeks)
Critical: No validation prevents provider upgrades that break connected clients.

---

## Version Compatibility Analysis

### Current State

**Version Checks Identified:**

| Location | Current Behavior | Required Change |
|----------|-----------------|-----------------|
| **server.go:236** | ❌ Rejects if major.minor don't match | ✅ Allow n-4 window (v4.20-v5.1) |
| server.go:3085 | ✅ Rejects if client ahead of server | ✅ Keep (correct) |
| server.go:370 | ⚠️ Version-specific CSI API workaround | ✅ Expand pattern |

### Version Support Matrix for v5.1

| Client Version | Onboarding | Features | DR | Notes |
|----------------|------------|----------|-----|-------|
| v5.1           | ✅         | All      | ✅  | Full support |
| v5.0 (4.23)    | ✅         | All      | ✅  | Full support |
| v4.22          | ✅         | Most     | ❌  | No NVMeOF/VAC |
| v4.21          | ✅         | Most     | ❌  | No NVMeOF/VAC |
| v4.20          | ✅         | Basic    | ❌  | Basic only |
| **v4.19 and older** | ❌     | None     | ❌  | **UNSUPPORTED** |

### Why v4.19 Cannot Be Supported

**Commit `6020825a9`** made two breaking changes:

1. **Reserved field 1** in `GetDesiredClientStateResponse`
   - Old clients expected `repeated bytes kubeResources = 1`
   - Now `reserved 1` - field is empty
   - Clients v4.19 and below cannot receive resources

2. **Removed 5 RPCs**
   - GetStorageConfig, AcknowledgeOnboarding, FulfillStorageClaim, RevokeStorageClaim, GetStorageClaimConfig
   - Old clients calling these get "Unimplemented" error

**Decision:** Accept v4.20 as minimum (these changes happened before v4.20)

---

## Critical Issues & Blockers

### Status Summary

| Category | Count | Status |
|----------|-------|--------|
| Critical Blockers | 1 | Must fix for n-4 |
| High Priority | 7 | Should fix |
| Medium Priority | 6 | Nice to have |
| Constraints (accepted) | 4 | Document only |

### Critical Blockers (Must Fix)

#### 1. Hard Version Rejection in OnboardConsumer
**File:** `services/provider/server/server.go:236`  
**Severity:** 🔴 CRITICAL

**Current Code:**
```go
if serverVersion.Major != clientVersion.Major || serverVersion.Minor != clientVersion.Minor {
    return nil, status.Errorf(codes.FailedPrecondition, 
        "both server and client operators major and minor versions should match")
}
```

**Issue:** Rejects ANY version mismatch  
**Impact:** Blocks all n-4 support  
**Fix:** Replace with configurable n-4 window check (see Document 3)

**Effort:** 3 story points

---

### High Priority Issues

#### 2. No Provider Upgrade Validation
**Severity:** 🔴 CRITICAL (Operational)

**Problem:**
```
Provider v5.1 → v5.2 upgrade
Has clients: v4.20, v4.21, v5.0
v5.2 minimum: v4.22 (n-4)
❌ Clients v4.20, v4.21 break silently!
```

**Impact:** Provider upgrades can silently break connected clients  
**Fix:** Pre-upgrade validation + admission webhook (Document 3)  
**Effort:** 20 story points

#### 3. No Capability Negotiation
**Severity:** 🟠 HIGH

**Issue:** Client sends only version, no feature declaration  
**Impact:** Can't handle backported features (e.g., v4.22.5 with enhanced-auth)  
**Fix:** Capability-based negotiation (Document 2)  
**Effort:** 13 story points

#### 8. No Upgrade Compatibility Matrix in Operator
**Severity:** 🟠 HIGH

**Issue:** Operator doesn't know its own compatibility matrix  
**Impact:** Can't validate upgrades programmatically  
**Fix:** Embed matrix in operator code (Document 3)  
**Effort:** 2 story points

---

## Implementation Roadmap

### Phase 1: Proto Separation (4 weeks)
Extract proto to `odf-provider-spec` repository.

**Tasks:**
- Week 1: Create repo, copy files, tag v1.0.0
- Week 2: Write SPEC.md, COMPATIBILITY.md
- Week 3: Migrate ocs-operator
- Week 4: Migrate ocs-client-operator

**Deliverable:** Both operators using `github.com/red-hat-storage/odf-provider-spec/lib/go/provider/v1`

### Phase 2: Capability Negotiation (3 weeks)
Add capability-based feature detection.

**Tasks:**
- Week 1: Define proto Capability message
- Week 2: Server capability inference + storage
- Week 3: Client capability detection

**Deliverable:** Capability negotiation working (Document 2)

### Phase 3: Configurable Version Check (1 week)
Make version compatibility configurable.

**Tasks:**
- Week 1: Environment variables, validation, metrics

**Deliverable:** Configurable version policy (Document 3)

### Phase 4: Provider Upgrade Safety (3 weeks)
Prevent breaking upgrades.

**Tasks:**
- Week 1: Metrics & alerts
- Week 2: Pre-upgrade validation
- Week 3: Admission webhook + docs

**Deliverable:** Safe provider upgrades (Document 3)

### Phase 6: Testing & Documentation (2 weeks)
Comprehensive testing and docs.

**Tasks:**
- Week 1: Test matrix, integration tests
- Week 2: Documentation, examples, runbooks

**Deliverable:** Production-ready n-4 support

---

## Effort Estimates

| Component | Story Points | Risk |
|-----------|--------------|------|
| **Phase 1:** Proto separation | 8 | Low |
| **Phase 2:** Capability negotiation | 13 | High |
| **Phase 3:** Configurable version check | 3 | Low |
| **Phase 4:** Provider upgrade safety | 20 | Medium |
| **Phase 6:** Testing & documentation | 10 | Medium |
| **Total** | **59 SP** | |

**Timeline:** 14 weeks (~3.5 months)  
**Team:** 2 engineers  
**Risk Level:** Medium

---

## Success Criteria

### Must Have (v5.1 Release)

- ✅ Provider v5.1 accepts clients v4.20 - v5.1
- ✅ Provider v5.1 rejects clients < v4.20 and > v5.1
- ✅ DR peering enforces version match
- ✅ All existing functionality works for v5.1 clients
- ✅ Basic functionality works for v4.20 clients
- ✅ Provider upgrade validation prevents breaking clients
- ✅ Documentation complete

### Should Have (v5.1.1)

- ✅ Capability-based feature detection
- ✅ Automated compatibility testing
- ✅ Telemetry for version skew

---

## Risks & Mitigation

### High Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Breaking changes limit backward compat | Can't support v4.19- | 100% | Accept limitation, document clearly |
| Testing matrix explosion | Quality issues | High | Automated CI for all combinations |
| Provider upgrade breaks clients | Production outage | High | Pre-upgrade validation (Phase 4) |

### Medium Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Feature parity tracking complexity | Bugs | Capability registry |
| Performance degradation | Slower reconciliation | Optimize hot paths |
| Documentation drift | Support overhead | Automated compatibility docs |

---

## Recommendations

### Immediate (Start Now)

1. ✅ **Proto separation** - Ready, low risk, foundation for rest
2. ✅ **Document compatibility policy** - Prevent future breaking changes
3. ✅ **Provider upgrade validation** - Critical safety gap

### Short Term (v5.1)

4. ✅ **Capability negotiation** - Foundation for n-4
5. ✅ **Configurable version check** - Operational flexibility
6. ✅ **DR version check** - Safety improvement

### Long Term (v5.2+)

7. ⚠️ **Expand to n-5** - After n-4 validation
8. ⚠️ **Multi-language proto bindings** - Ecosystem growth
9. ⚠️ **Migration tooling** - User experience

---

## Decision Matrix

| Option | Effort | Risk | Benefits | Recommendation |
|--------|--------|------|----------|----------------|
| **Proto separation** | Low (4w) | Low | Foundation, multi-lang | ✅ DO IT |
| **n-4 support** | High (14w) | Medium | Customer compatibility | ✅ DO IT |
| **Provider upgrade safety** | Medium (3w) | Low | Prevent outages | ✅ DO IT |

---

## Honest Assessment

### ❓ Can we support n-4/n-5 TODAY?
**❌ NO** - Breaking changes in v4.20 prevent supporting v4.19 and below

### ❓ Can we support n-4/n-5 GOING FORWARD?
**✅ YES** - But with constraints:
- Minimum v4.20 (cannot go lower)
- Requires 14 weeks engineering
- Requires strict proto evolution policy
- High testing complexity
- **Critical:** Need provider upgrade safety

### ❓ Should we proceed?
**✅ YES, RECOMMEND:**
1. Proto separation (4 weeks)
2. Provider upgrade safety (3 weeks) 
3. n-4 support (14 weeks total)
4. DR version check (1 week)

---

## Next Actions

### This Week
1. Review analysis with team
2. Get approval for `odf-provider-spec` repository
3. Prioritize: Proto separation + upgrade safety

### Next Sprint
1. Create `odf-provider-spec` repository
2. Implement provider upgrade validation
3. Start capability negotiation design

### Within 1 Month
1. Complete proto separation
2. Provider upgrade safety operational
3. Begin n-4 implementation

---

**For detailed designs, see:**
- Document 2: Proto & Capability Design
- Document 3: Version Configuration & Upgrade Safety
