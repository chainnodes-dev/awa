import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Chain Nodes",
  description: "Enterprise-grade Autonomous Agent Platform",
  ignoreDeadLinks: true,
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/assets/logos/favicon.svg' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    ['link', { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=Outfit:wght@400;600;800;900&display=swap' }]
  ],
  themeConfig: {
    logo: '/assets/logos/favicon.svg',
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Comparison', link: '/comparison' },
      { text: 'Pricing', link: '/pricing' },
      { text: 'GitHub', link: 'https://github.com/chainnodes-dev/awa.git' }
    ],
    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Architecture', link: '/guide/architecture' },
          { text: 'Chain Nodes vs. Competitors', link: '/comparison' },
          { text: 'Pricing & Features', link: '/pricing' },
          { text: 'State Nodes', link: '/guide/nodes' },
          { text: 'Transitions', link: '/guide/transitions' },
          { text: 'UI Documentation', link: '/guide/ui' },
        ]
      },
      {
        text: 'Advanced',
        items: [
          { text: 'MCP Integration', link: '/guide/mcp' },
          { text: 'Workflow DSL', link: '/guide/workflows' },
        ]
      },
      {
        text: 'API Reference',
        items: [
          { text: 'Overview', link: '/guide/api' },
          { text: 'YAML Schema', link: '/guide/yaml-schema' },
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/chainnodes-dev/awa.git' },
      { icon: 'discord', link: 'https://discord.gg/phaxa' }
    ],
    footer: {
      message: 'Released under the Enterprise License. Chain Nodes S.R.L. | Str. Brizei nr. 8, Cluj-Napoca, Romania | CUI: 39890133 | Reg: J12/4187/2018',
      copyright: 'Copyright © 2024-present Chain Nodes S.R.L.'
    }
  }
})
