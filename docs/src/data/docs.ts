export const siteConfig = {
  name: "Noria",
  strapline: "Keep your agents in flow",
  description:
    "Set up your Codex and Claude account pool once. Use your normal clients, manage usage from the native menu bar, and let Noria handle routing.",
  repoUrl: "https://github.com/amxv/noria",
  logoHref: "/favicon.svg",
  accentColor: "#8e3d25",
  accentColorDark: "#e7a18a",
  footerSections: [
    {
      title: "On your machine",
      text:
        "A local Go gateway. Your native clients, your authorized accounts, your requested model. No cloud service in the routing path."
    },
    {
      title: "Built in the open",
      text:
        "v0.2.0 · Apache-2.0. Native binaries for macOS, Linux, and Windows. No paid fallback or model swaps."
    },
    {
      title: "Source & releases",
      linkHref: "https://github.com/amxv/noria",
      linkLabel: "github.com/amxv/noria"
    }
  ]
} as const;

export const docCategories = [
  "Get started",
  "Use Noria",
  "Reference"
] as const;

export const primaryNav = [
  { href: "/docs", label: "Docs" },
  { href: "/docs/scheduling", label: "How it routes" },
  { href: siteConfig.repoUrl, label: "GitHub", external: true }
];
