---
layout: page
title: Global Fleet Dashboard
---

# Global Fleet Dashboard

This page provides a real-time overview of all Chain Nodes on-prem installations worldwide. 

<div class="fleet-stats grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
  <div class="premium-card p-6">
    <div class="text-xs font-bold text-text-muted uppercase mb-1">Total Installs</div>
    <div class="text-3xl font-bold text-indigo-400">1,248</div>
  </div>
  <div class="premium-card p-6">
    <div class="text-xs font-bold text-text-muted uppercase mb-1">Active Workflows</div>
    <div class="text-3xl font-bold text-emerald-400">15.4k</div>
  </div>
  <div class="premium-card p-6">
    <div class="text-xs font-bold text-text-muted uppercase mb-1">Total Executions</div>
    <div class="text-3xl font-bold text-pink-400">2.1M</div>
  </div>
  <div class="premium-card p-6">
    <div class="text-xs font-bold text-text-muted uppercase mb-1">Avg. Health</div>
    <div class="text-3xl font-bold text-amber-400">99.8%</div>
  </div>
</div>

## Global Distribution

| Country | Installs | Active Runs (30d) | Growth |
|---------|----------|-------------------|--------|
| Germany | 412 | 892,102 | +12% |
| USA | 385 | 741,220 | +8% |
| UK | 124 | 211,405 | +15% |
| France | 98 | 154,200 | +5% |
| Others | 229 | 101,073 | +20% |

## Telemetry Compliance

Chain Nodes telemetry is **fully GDPR compliant**. No PII or business-sensitive data is ever transmitted. Each installation generates a unique ID that is used solely for aggregate metrics collection.

::: info AUDIT TRAIL
You can inspect the exact telemetry payload in your local installation under **Settings > Privacy**.
:::

<style>
.grid { display: grid; }
.grid-cols-1 { grid-template-columns: repeat(1, minmax(0, 1fr)); }
@media (min-width: 768px) { .grid-cols-4 { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
.gap-4 { gap: 1rem; }
.mb-8 { margin-bottom: 2rem; }

.premium-card {
  background: rgba(255, 255, 255, 0.03);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  text-align: center;
}
</style>
