---
title: How routing helps
description: Warm sessions stay put; new work uses allowance that is useful now.
order: 5
category: Use Noria
summary: Warm sessions stay put; new work uses allowance that is useful now.
---

Noria makes the account choice so you do not have to watch several reset clocks while coding.

## Stay with the warm account

A session stays on its account while that account can serve it. Noria does not shuffle an active conversation because another account has more headroom or has just reset. This preserves the opportunity to reuse its account-local prompt cache.

When an applicable limit is exhausted, Noria retries the unfulfilled model request on another eligible account. The session stays there afterward; it does not bounce back. The new account's cache warms through normal requests.

## Use the right remaining allowance

New sessions and necessary failovers consider each account's independent five-hour and weekly windows, model-specific constraints, work already assigned, and observed consumption.

An account with **5% weekly remaining and a reset soon** can be more useful for new work than one with 80% remaining and several days left. But existing sessions on the second account stay there. The scheduler also avoids sending every new agent to an expiring allowance that existing work is already expected to consume.

Percentages in different allowance windows are not equal units of work. The scheduler uses per-window estimates rather than blindly taking the smaller percentage.

## Keep your requested behavior

Your provider, model, settings, tool events and request content are not translated into a cheaper alternative. Noria does not buy extra credits, redeem reset credits, or start dummy prompts to activate windows.

If every eligible account is unavailable, Noria reports that and any known next usable reset. A five-hour reset alone does not help an account whose weekly allowance is still exhausted.

The placement policy is a bounded heuristic, not a guarantee that every unit of quota can always be spent. Hard affinity deliberately wins over chasing another account's expiring quota. More independent sessions give the router more placement opportunities.

See [Architecture and limits](/docs/architecture) for the precise control and retry boundaries.
