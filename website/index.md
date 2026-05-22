---
layout: home

# We disable the default hero so we can use our custom HomeHero component
hero:
  name: "Chain Nodes"
  text: ""
  tagline: ""

features:
  - title: "AI-Native Synthesis"
    details: "Agentic blueprinting for business logic. Describe your requirements in plain English and Chain Nodes automatically scaffolds high-fidelity state machines."
  - title: "Durable State Machines"
    details: "Beyond simple DAGs. Chain Nodes handles loops, multi-day pauses, and deep retries with surgical precision. Zero-loss execution for critical logic."
  - title: "MCP Tool-Space"
    details: "Native Model Context Protocol support. Your agents can securely browse docs, query SQL, or push to GitHub using the latest AI standards."
---

<HomeHero />

<div class="features-divider"></div>

## Why Chain Nodes?

Chain Nodes is designed for corporate environments where "black box" agents aren't enough. By combining **Temporal.io** for reliability with the **Model Context Protocol (MCP)** for connectivity, Chain Nodes allows you to build workflows that are both autonomous and auditable.

::: tip PRO TIP
Use our **AI Generator** in the dashboard to scaffold your first workflow by simply describing your business process in plain English.
:::

<ContactForm />

<style>
.features-divider {
  height: 1px;
  background: linear-gradient(to right, transparent, var(--vp-c-brand), transparent);
  margin: 64px 0;
  opacity: 0.3;
}

:deep(.VPFeatures) {
  padding: 0 24px;
}

:deep(.VPFeature) {
  background: var(--vp-c-bg-soft);
  backdrop-filter: blur(12px);
  border: 1px solid var(--vp-c-divider);
  border-radius: 24px;
  transition: all 0.3s ease;
  padding: 32px;
}

:deep(.VPFeature h2) {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 16px;
  color: var(--vp-c-text-1);
}

:deep(.VPFeature:hover) {
  border-color: var(--vp-c-brand);
  background: var(--vp-c-bg-mute);
  transform: translateY(-4px);
}
</style>
