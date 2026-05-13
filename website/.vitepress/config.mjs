import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "Phaxa",
  description: "Enterprise Agentic Workflow Platform",
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'API', link: '/reference/api' },
      { text: 'Dashboard', link: 'http://localhost:5173' }
    ],
    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Architecture', link: '/guide/architecture' },
        ]
      },
      {
        text: 'Core Concepts',
        items: [
          { text: 'Workflow DSL', link: '/guide/workflows' },
          { text: 'MCP Integration', link: '/guide/mcp' },
          { text: 'Human-in-the-Loop', link: '/guide/hitl' },
        ]
      },
      {
        text: 'API Reference',
        items: [
          { text: 'Overview', link: '/reference/api' },
          { text: 'YAML Schema', link: '/reference/yaml-schema' },
        ]
      }
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/phaxa-ai' }
    ],
    footer: {
      message: 'Released under the Enterprise License.',
      copyright: 'Copyright © 2024-present Phaxa AI'
    }
  }
})
