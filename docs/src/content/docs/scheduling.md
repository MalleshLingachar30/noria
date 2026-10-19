---
title: Routing algorithm
description: Hard session affinity, reset-aware placement, and honest behavior when capacity runs out.
order: 4
category: Routing
summary: How Noria chooses an account, keeps a session there, and handles independent quota clocks.
---

Noria makes account placement decisions when a session starts and when its account becomes unavailable. It does not rotate healthy sessions on each request.

## Hard session affinity

Once the upstream accepts work, Noria binds the session to that account. It stays there while the account is enabled, matches the provider and model, and has no applicable blocker. A better placement score, another account’s reset, or low remaining quota alone does not evict it.

After quota forces a switch and the new account accepts work, the session stays with the new account. It does not bounce back when the old allowance resets. A temporary network or throughput failure is handled separately and should not permanently move a healthy binding. Removing or disabling an account makes it ineligible.

Bindings and blocker metadata persist across daemon restarts. New branches with their own session identity can be placed independently. Requests without a recognizable session identity do not all share a synthetic global session; they cannot receive the same cross-request affinity guarantee. See [native clients](/docs/native-clients).

## Two clocks, on every account

Each account’s five-hour and weekly windows are independent. A usable five-hour window cannot override an exhausted week. Model-scoped limits add their own constraints. Every applicable blocker must clear before that account is usable again.

Provider utilization means **percent used**. Examples below show **percent remaining** for readability. Missing telemetry, null resets, and absent windows remain unknown. The backend’s actual rejection is authoritative even when the last display looked healthy.

Noria does not send dummy prompts to activate windows. A missing reset stays unknown unless an adapter explicitly identifies the window as awaiting first use. Only then can accepted real work start a predicted clock. An authoritative returned reset takes precedence. A prediction is not a provider guarantee.

## Placement at the right moments

For initial placement and failover, the scheduler:

1. Filters to enabled accounts for the requested provider and exact model eligibility.
2. Checks applicable quota, authentication, and cooldown constraints.
3. Keeps a healthy existing binding before considering a new score.
4. Estimates useful remaining capacity in each window using window-specific learned consumption, then accounts for expected work before reset, in-flight requests, and the recent cadence of already-bound sessions.
5. Selects and reserves under one lock, so simultaneous requests see work already assigned.

This is a bounded expiry and capacity heuristic, not a proof of globally optimal scheduling. Percentages across different plans or window durations are not interchangeable units of work. Learning is based on observed consumption per completed request, so early estimates are uncertain and workloads that vary greatly can weaken the forecast.

Reservations represent estimated work already in flight. Completion and fresh telemetry reconcile those estimates without counting the same work twice. They influence placement rather than impose a fixed unused allowance: there is no arbitrary 5% safety reserve or automatic eviction at 90% or 95% usage.

## An illustrative placement

These values are invented to explain the decision, not live telemetry or a promised selection.

| Account | Five-hour remaining | Weekly remaining | Current condition |
| --- | --- | --- | --- |
| A | 72%, resets in 3h | 5%, resets in 18m | Candidate for new work |
| B | 84%, resets in 4h | 61%, resets in 4d | Healthy bound sessions stay |
| C | 90%, resets in 2h | Exhausted, resets in 6h | Blocked by weekly quota |

A may be useful now because its remaining weekly allowance expires soon. That preference depends on expected work and reservations; a burst of new sessions should not all assume A’s allowance is theirs. B’s existing sessions stay on B. C’s five-hour headroom does not clear its weekly blocker.

Hard affinity has a deliberate tradeoff: one long, healthy session may leave allowance unused on other accounts. Independent sessions can use a wider pool. Noria does not move an active session simply to spend spare quota.

## Unknown telemetry and exhaustion

Fresh known-usable candidates take priority for new placements. When none exist, Noria permits a bounded, cautious attempt on an otherwise eligible account with unknown or stale telemetry. Unknown never means “100% free” or “exhausted.” An authentication rejection or unresolved quota blocker is not erased by a failed read.

The scheduler remains work conserving: it does not deliberately pause while an authorized matching account can serve. When no account is eligible, Noria returns unavailability rather than retrying forever. If reset information permits, it forecasts the earliest time at which **all** blockers on at least one account could clear. That is not necessarily the earliest individual clock in the pool. Unknown reset times may require a fresh quota read or account intervention instead of a countdown.

## Failover and the stream boundary

Noria retries the pending native request on another authorized matching account when safe, retaining its model, settings, body, and continuation data. It distinguishes quota exhaustion from authentication failure, transient throughput limits, network failures, nonretryable client errors, and provider-wide server errors.

Safe retries are bounded and happen before output is committed, including recognized early quota errors in an SSE stream. After content has been delivered, Noria cannot transparently splice a replacement generation into it or replay tool output safely. An interrupted stream remains visible to the client. The client or user must decide how to continue.

## Account-local caches

Affinity preserves the opportunity to reuse an account’s warm upstream cache. After a forced switch, the replacement account starts cold for that context and warms through normal work. Noria does not copy caches, migrate transcripts, or guarantee a cache hit. A provider may expire its own cache independently.

Noria never swaps provider or model, buys extra credits, redeems reset credits, or creates capacity when every account is exhausted.
