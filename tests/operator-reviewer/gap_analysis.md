# Sprint 6 Gap Analysis: `operator-reviewer` Subagent

## What the Reviewer Checks

The reviewer combines automated script validation with manual pattern inspection.

### Automated (via existing skill scripts)

| Check | Script | Findings |
|-------|--------|----------|
| API types: markers, json tags, conditions, print columns | validate-api-types.py | 14 structural checks |
| RBAC: least privilege, no wildcards, matching resources | validate-rbac-annotations.py | Marker consistency |
| Idempotency: Get before Create, owner refs, events | check-idempotency.py | Pattern compliance |

### Manual (agent inspection)

| Check | Category | What It Catches |
|-------|----------|-----------------|
| Finalizer lifecycle completeness | Critical | Missing re-fetch, incomplete cleanup |
| Error path condition updates | Warning | Status doesn't reflect error state |
| Dependency ordering | Warning | Resources created in wrong order |
| Hardcoded values | Warning | Values that should come from CR spec |

## Comparison: Reviewer vs Manual Code Review

| Aspect | Manual Review | Operator-Reviewer |
|--------|---------------|-------------------|
| Time | 30-60 min per operator | 2-5 min |
| Consistency | Varies by reviewer | Same checklist every time |
| Automated checks | None (reading only) | 3 scripts with ~30 checks |
| False positives | Rare | Rare (scripts are well-tested) |
| Subtle issues | Catches context-dependent bugs | May miss complex interaction bugs |
| Fix suggestions | General advice | Specific references to skill patterns |

## What the Reviewer Does NOT Check

| Area | Why Not | Where It's Checked |
|------|---------|-------------------|
| Bundle/CSV correctness | Separate concern | operator-bundle-validator (Sprint 8) |
| Test coverage | Separate concern | operator-test-generator (Sprint 7) |
| Scaffolding structure | Already validated at scaffold time | scaffolding-operator script |
| Runtime behavior | Needs real cluster | E2E tests (post-Sprint 8) |
| Performance | Needs profiling | Not in scope |

## Summary

The reviewer automates ~80% of what a senior engineer checks during a controller code review. The remaining 20% (complex interaction bugs, performance, runtime behavior) requires manual investigation or E2E testing. The key value is **consistency** — every operator gets the same thorough checklist applied.
