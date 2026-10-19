---
title: "Security and limits"
description: "Separate credentials, local access controls, and compatibility boundaries."
order: 7
category: "Reference"
summary: "Separate credentials, local access controls, and compatibility boundaries."
---

Noria is a local router, not local inference. Model traffic still goes to the selected provider.

Pool sign-ins are isolated. Client integration installs a local credential helper and backs up routing values; it does not copy pool credentials into your normal client login. Stop restores managed direct routing. Profiles with manually pasted settings and Claude Desktop's separate inference form are not automatically restored.

The loopback gateway checks its random credential, Host, and Origin. This does not protect against malicious software already running as your operating-system user. Keep the private home secure and do not expose the gateway through a public tunnel.

Noria does not create quota, purchase credits, redeem resets, switch models/providers, move caches, or splice already-delivered output. Updates and health checks do not make model requests.

Remove unregisters an account but retains native profile files for recovery. A complete provider logout may also involve the platform's credential store. Never delete unrelated client profiles or Keychain entries.

## Distribution

The macOS app is ad-hoc signed and installed through a checksum-verified CLI download/extract flow, matching Zodex. No installer disables Gatekeeper or removes quarantine metadata. Browser-downloaded bundles and managed-device policies can behave differently. This is not a Developer ID notarization claim.

## Compatibility

Native quota/auth interfaces can change. Tests use isolated profiles and controlled upstreams, not live subscription allowances. Actual cache hits and authenticated backend acceptance still require real authorized use. Existing clients may cache their routing and need reopening after changes. Provider use remains subject to the applicable account terms.
