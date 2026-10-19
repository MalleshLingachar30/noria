export const siteConfig = {
  name: 'Noria',
  strapline: 'Smart account routing',
  description:
    'Use more of your Codex and Claude Code limits without constantly switching accounts. A smart proxy that runs locally.',
  repoUrl: 'https://github.com/amxv/noria',
  logoHref: '/favicon.svg',
  accentColor: '#2858e6',
  accentColorDark: '#94b5ff',
  footerSections: [
    {
      title: 'Built for your workflow',
      text: 'A smart local proxy for the coding accounts you already use. No cloud routing control plane.'
    },
    {
      title: 'Open source',
      text: 'v0.3.2 · Apache-2.0 · macOS, Linux, and Windows.'
    },
    {
      title: 'Code & releases',
      linkHref: 'https://github.com/amxv/noria',
      linkLabel: 'github.com/amxv/noria'
    }
  ]
} as const;

export const docCategories = ['Get started', 'How it works', 'Reference'] as const;

export const primaryNav = [
  { href: '/docs', label: 'Docs' },
  { href: '/docs/architecture', label: 'How it works' },
  { href: siteConfig.repoUrl, label: 'GitHub', external: true }
];
