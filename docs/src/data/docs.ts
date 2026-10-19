export const siteConfig = {
  name: "Noria",
  strapline: "Keep your agents in flow",
  description:
    "Set up once and use Codex or Claude Code normally. Noria handles account routing, with native macOS menu controls and background service support.",
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
        "v0.3.1 · Apache-2.0. Native installers for macOS, Linux, and Windows. No paid fallback or model swaps."
    },
    {
      title: "Source & releases",
      linkHref: "https://github.com/amxv/noria",
      linkLabel: "github.com/amxv/noria"
    }
  ]
} as const;

export const docCategories = [
  "Start",
  "Routing",
  "Reference"
] as const;

export const primaryNav = [
  { href: "/docs", label: "Docs" },
  { href: "/docs/scheduling", label: "How it routes" },
  { href: siteConfig.repoUrl, label: "GitHub", external: true }
];
