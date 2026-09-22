# ODF n-4/n-5 Version Compatibility Analysis

**Date:** 2026-09-16  
**For:** ODF v5.1 Release Planning

---

## 📚 Document Set (3 Documents)

| # | Document | Size | Purpose | Read Time |
|---|----------|------|---------|-----------|
| **1** | [ANALYSIS_AND_FEASIBILITY.md](01_ANALYSIS_AND_FEASIBILITY.md) | 12KB | Complete analysis, bugs, roadmap | 15 min |
| **2** | [PROTO_AND_CAPABILITY_DESIGN.md](02_PROTO_AND_CAPABILITY_DESIGN.md) | 16KB | Proto separation + capability negotiation | 20 min |
| **3** | [VERSION_CONFIGURATION_AND_UPGRADE_SAFETY.md](03_VERSION_CONFIGURATION_AND_UPGRADE_SAFETY.md) | 16KB | Configurable version policy + upgrade safety | 20 min |

---

## 🎯 Quick Summary

### ✅ Proto Separation (4 weeks)
Extract proto to `odf-provider-spec` repository (like csi-addons/spec).  
**Status:** Ready to execute - 90% done

### ⚠️ n-4/n-5 Support (14 weeks)
Support clients v4.20 through v5.1 for provider v5.1.  
**Status:** Feasible with 1 critical blocker

### 🔴 Upgrade Safety (3 weeks)
Prevent provider upgrades that break connected clients.  
**Status:** Critical gap - must implement

---

## 🚀 Quick Start

### If you have 5 minutes:
Read this page + Executive Summary in Document 1

### If you have 30 minutes:
Read Document 1 completely

### If you have 1 hour:
Read all 3 documents

### If you need specifics:
- **Bugs/issues?** → Document 1, section "Critical Issues"
- **Proto changes?** → Document 2, section "Proto Design"
- **Version policy?** → Document 3, section "Configurable Version Check"
- **Upgrade safety?** → Document 3, section "Provider Upgrade Safety"

---

## 📊 Key Decisions Made

| Decision | Rationale |
|----------|-----------|
| **Minimum v4.20** | Breaking changes in commit `6020825a9` prevent v4.19 support |
| **Hybrid capabilities** | Client advertises, server infers for old clients |
| **Configurable via env** | Version policy adjustable without code changes |
| **Pre-upgrade validation** | Block provider upgrades that would break clients |
| **DR same-version** | Too risky to support version skew for DR |

---

## 🔢 Numbers

| Metric | Value |
|--------|-------|
| **Total effort** | 59 story points |
| **Timeline** | 14 weeks |
| **Critical blockers** | 1 (hard version check) |
| **High priority issues** | 7 |
| **Supported client range** | v4.20 - v5.1 |
| **Documents** | 3 (was 7) |

---

## ⚡ Critical Blocker

**Only 1 blocker prevents n-4 support:**

```go
// services/provider/server/server.go:236
if serverVersion.Major != clientVersion.Major || 
   serverVersion.Minor != clientVersion.Minor {
    return nil, status.Errorf(codes.FailedPrecondition, 
        "versions must match")  // ← THIS
}
```

**Fix:** Replace with configurable n-4 window check (3 SP effort)

---

## 🎯 Recommended Action Plan

### Phase 1: Foundation (5 weeks)
1. Proto separation (4 weeks)
2. Configurable version check (1 week)

### Phase 2: Safety (3 weeks)
3. Provider upgrade validation (3 weeks)

### Phase 3: Implementation (6 weeks)
4. Capability negotiation (3 weeks)
5. DR version check (1 week)
6. Testing & docs (2 weeks)

**Total: 14 weeks**

---

## 💡 Key Insights

### What Works
✅ Provider API already has separate go.mod  
✅ Proto uses optional fields (backward compatible)  
✅ Version stored in StorageConsumer.Status  
✅ Clear version boundary at v4.20  

### What's Missing
🔴 No provider upgrade validation  
🔴 Hard version check blocks n-4  
🔴 No capability negotiation  
🔴 No client version metrics  

### What Changed
- **Before analysis:** Thought v4.19 support needed → 3 blockers
- **After analysis:** Accept v4.20 minimum → 1 blocker ✅
- **Effort reduced:** 55 SP → 59 SP (added upgrade safety)

---

## 📋 Version Support Matrix

| Client | Provider v5.1 | Features | DR |
|--------|---------------|----------|-----|
| v5.1 | ✅ Full | All | ✅ |
| v5.0 | ✅ Full | All | ✅ |
| v4.22 | ✅ Partial | Most | ❌ |
| v4.21 | ✅ Partial | Most | ❌ |
| v4.20 | ✅ Basic | Basic | ❌ |
| v4.19- | ❌ None | None | ❌ |

---

## 🛠️ Implementation Checklist

### Document 1 Actions
- [ ] Review and approve analysis
- [ ] Accept v4.20 as minimum
- [ ] Prioritize 18 identified issues

### Document 2 Actions
- [ ] Create `odf-provider-spec` repository
- [ ] Define `Capability` proto message
- [ ] Implement capability inference

### Document 3 Actions
- [ ] Add environment variables for version config
- [ ] Implement pre-upgrade validation
- [ ] Add Prometheus metrics and alerts

---

## 📞 Questions?

**For technical details:** Read the specific document section  
**For implementation:** See roadmap in Document 1  
**For proto changes:** See Document 2  
**For operational concerns:** See Document 3

---

## ✨ Bottom Line

**n-4/n-5 version compatibility is achievable in 14 weeks** with these constraints:

- ✅ Minimum v4.20 (technical constraint from past breaking changes)
- ✅ Proto separation ready (4 weeks)
- ✅ Only 1 critical blocker to fix (3 story points)
- 🔴 Must implement upgrade safety (critical operational gap)
- ⚠️ Requires strict proto evolution policy going forward

**Recommendation:** Proceed with implementation.

---

**Analysis Date:** 2026-09-16  
**Valid For:** ODF v5.1 planning  
**Next Review:** After proto separation completion
