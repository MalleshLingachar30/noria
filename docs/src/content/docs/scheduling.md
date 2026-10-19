---
title: "How routing helps"
description: "Sticky sessions and reset-aware placement without manual account switching."
order: 5
category: "Routing"
summary: "Sticky sessions and reset-aware placement without manual account switching."
---

## One session stays on one account

Noria chooses an account when a session starts. Requests remain on that account while it can serve. Another account resetting or getting a better placement score never moves a healthy session.

This preserves account-local prompt-cache reuse. If an applicable five-hour, weekly, or model limit blocks the next request, Noria chooses another eligible account, retries the pending native request, and stays on the new account. Its cache warms naturally. Noria does not bounce back when the original account resets.

## Use capacity before it expires

Every account has its own five-hour and weekly reset clocks. A small remaining allowance that resets soon can be more useful now than a large allowance with days left.

These values are illustrative:

| Account | Weekly remaining | Resets in | Use |
| --- | --- | --- | --- |
| A | 5% | 20 minutes | Useful candidate for new work before expiry |
| B | 80% | 4 days | Preserve for later work; keep its existing sessions |
| C | Exhausted | 2 hours | Blocked even if its five-hour allowance is healthy |

Placement accounts for deadlines, reservations, and recent bound-session demand. Consumption is estimated separately for each quota window; five-hour and weekly percentages are not interchangeable work units.

This is a bounded heuristic, not a guarantee of globally optimal usage. One long sticky session can leave spare quota on other accounts unused. Noria deliberately avoids moving that session just to rebalance capacity.

## At a limit

Safe retry preserves the original provider, model, settings, and conversation data. Recognized quota errors are retried before useful output is delivered. Noria does not replay completed tool actions or splice an already-delivered partial response into a replacement generation.

All applicable limits must permit an account to serve. A five-hour reset cannot clear an exhausted week. Unknown telemetry stays unknown, and an actual upstream rejection is authoritative.

If no matching account is usable, Noria reports the shortage and the next usable reset when known. It does not purchase capacity, redeem reset credits, or switch models/providers to conceal exhaustion.
