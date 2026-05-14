# Go-to-Market Plan
## Agentic Workflow Automation Platform (Phaxa)
*Bootstrapped · Stealth-to-Launch · Europe-first*
*Updated: 2026-05-13 — reflects current codebase state*

---

## Codebase Status vs. Pre-Launch Checklist

The following items from the original plan were audited against the repo on 2026-05-13:

| Item | Status | Notes |
|---|---|---|
| `docker-compose.yml` single-command stack | ✅ Done | Both `docker-compose.yaml` (dev) and `docker-compose.prod.yaml` present |
| README with 5-min quick start | ✅ Done | `make stack` brings up full stack |
| Enterprise feature code (SSO, RBAC, audit, license key) | ✅ Done | `internal/enterprise/` — `audit.go`, `license.go` in place |
| Website / landing page | ✅ Done | `website/` with `index.html` |
| Docs site structure | ✅ Done | `docs/` — architecture, API guide, user manual present |
| Frontend analytics store | ✅ Done | `frontend/src/stores/enterprise.js` |
| **License → Polyform Perimeter** | ✅ Done | Migrated from Polyform Noncommercial on 2026-05-13 |
| `CONTRIBUTING.md` | ✅ Done | Included in repository root |
| Hetzner / DigitalOcean one-click deploy | ✅ Done | Cloud-init and DO App spec added in `deploy/` |
| Opt-in server-side telemetry (install count, workflow count) | ✅ Done | Anonymous install beacon added on startup |
| Pricing page on website | ✅ Done | Available at `website/pricing.md` and in VitePress navigation |
| Discord server + link in README | ✅ Done | Added Discord badge and navigation links |
| Time-limited sandbox (48h, email-gated) | ❌ Missing | Phase 1 infrastructure |

---

## 1. Foundations

### 1.1 License Architecture

**Single license for the entire monorepo: PolyForm Perimeter 1.0.0**

Migrated from Polyform Noncommercial on 2026-05-13. Both `/LICENSE` (root) and
`/internal/enterprise/LICENSE` now reflect this.

| Who | What they can do |
|---|---|
| Any developer / company | Self-host, use internally, modify for internal use — free |
| Any company | Build workflows and automations on top of the platform — free |
| Any company | Use in production commercially for internal purposes — free |
| Nobody (without a commercial agreement) | Offer a product that competes with the platform (including re-hosting it as a service) |

**Key distinction from the previous Noncommercial license:** enterprises can now
freely evaluate and run the full product internally — including the enterprise
feature code — without hitting a license wall. The commercial agreement is solely
about the license key that activates enterprise features in production.

**Enterprise features are a product gate, not a license gate.** Enterprise features
(SSO/SAML, RBAC, audit logs, Git versioning, multi-tenant management) are unlocked
by a license key issued under a separate commercial agreement. The source code is
readable and auditable by anyone under Polyform Perimeter terms.

> **Note on Polyform Perimeter vs. Polyform Shield:** Perimeter prohibits any
> *competing product*, which is broader than just blocking SaaS re-hosting.
> Shield only blocks "delivering the functionality of the software to third parties."
> If the intent is purely to block re-hosting, revisit this before public launch.
> Perimeter is the more defensible choice if you also want to block embedded
> white-label competitors.

---

### 1.2 Positioning

**One-liner:** The open, self-hosted agentic workflow automation platform for
technical teams who need full control over their data, agents, and processes.

**What you are not:** A SaaS automation tool. A no-code toy. An n8n clone.

**Differentiation axes:**
- Agentic-first architecture built on Temporal — not retrofitted automation
- European-built: GDPR-native, data residency by default, on-premise friendly
- Polyform Perimeter: genuinely open for internal use, protected against clones

---

## 2. Pre-Launch Preparation (Weeks 1–4)
*Cost: ~€0 · No public activity yet*

### Repository & Code

- [x] `docker-compose.yaml` in root — single command brings up the full stack
- [x] README: one-liner, architecture reference, quick start
- [x] README: add honest comparison table vs. n8n, Temporal, Windmill, Power Automate
- [x] `CONTRIBUTING.md`: clarify `internal/` (private) vs. `pkg/` (public) vs. `internal/enterprise/` (enterprise); state what contributions go where
- [x] Opt-in telemetry: anonymous install count + active workflow count — server-side beacon, no PII, documented in README
- [x] Hetzner or DigitalOcean one-click deploy template

### Website & Content Infrastructure

- [x] Landing page (`website/index.html`) — exists, needs pricing anchor and email capture
- [x] Docs structure (`docs/`) — exists, may need a rendered site (Docusaurus or Mintlify)
- [x] Add pricing page to website (tier table — see Section 5)
- [ ] Blog (can live under docs initially)
- [ ] Discord server: create channels before launch — `#general`, `#show-and-tell`, `#integrations`, `#enterprise`
- [x] Add Discord link to README and website

### Content Pipeline

- [ ] 3-minute demo video: real workflow, real agents, real output
- [ ] 3 tutorial articles written and ready to publish at launch
- [ ] 3 YouTube tutorials scripted and filmed
- [ ] "Why we built this" founder story post

### Fast-Fail Gate

> Can 5 people from your network install and run a real workflow in under 10 minutes
> with no help from you?
> If no → fix onboarding first. Nothing else matters until this passes.

---

## 3. Phase 1 — Community Launch & Seeding (Months 1–3)
*Cost: €150–300/month · Team time: 2–3 hours/day on marketing*

### Launch Day Sequence

**Day 1:**
1. Push public GitHub repo
2. Post "Show HN" — Tuesday–Thursday, 9–11am US Eastern (2–4pm Frankfurt)
3. Reddit: r/selfhosted, r/devops, r/MachineLearning, r/artificial, r/LocalLLaMA
4. LinkedIn founder post: personal story, problem, link to GitHub
5. Twitter/X: thread version of LinkedIn post

**Show HN title:** `Show HN: Phaxa – Self-hosted agentic workflow automation (Polyform Perimeter)`

### Ongoing Content Cadence

| Channel | Frequency | Content type |
|---|---|---|
| LinkedIn | 3×/week | Use cases, lessons learned, ecosystem commentary |
| YouTube | 2×/month | Specific workflow tutorials (not product tours) |
| Reddit | 2×/month | Tutorial posts, answer questions in relevant subs |
| Twitter/X | Daily | Engage in agentic AI threads |
| Blog/Dev.to | 2×/month | Written tutorials, architecture deep-dives |
| Discord | Daily | Answer every question within 24 hours |
| GitHub | Weekly | Respond to every issue and PR within 48 hours |

**Content principle:** show a specific problem being solved, not the product being
demonstrated. "How we automated our invoice processing with 3 agents" outperforms
"introducing our platform" by 10:1 in organic reach.

### Infrastructure

- Time-limited sandbox: spin up on request, auto-destroy after 48 hours
  - Hetzner CX11 (~€4/month) can serve 20–30 concurrent instances
  - Activated by email signup — filters bots, builds the list
  - No credit card required

### Phase 1 Fast-Fail KPIs (End of Month 3)

| Metric | Minimum (continue) | Strong |
|---|---|---|
| GitHub stars | 500 | 1,500+ |
| Discord members | 150 | 400+ |
| Opt-in installs (telemetry) | 80 | 250+ |
| Inbound inquiries (email / Discord) | 10 | 30+ |
| YouTube subscribers | 100 | 500+ |

**If minimum thresholds not reached:** stop Phase 2, run a 2-week diagnosis sprint.
Interview 10 people who installed but didn't stay active. The problem is almost
always: wrong ICP, unclear differentiation, or onboarding friction.

---

## 4. Phase 2 — Design Partners (Months 3–6)
*Cost: €0 external · Revenue: €0 (intentional)*

### Goal

3–5 companies running the platform in production. At least 2 willing to pay for
enterprise features.

### Sourcing Design Partners

Use inbound from Phase 1:
- Enterprise-size Discord members asking "does this support SSO?"
- GitHub issues filed from company email domains
- Direct LinkedIn outreach to IT architects / heads of automation at ICP companies

**Offer:** hands-on onboarding, direct Slack/email access to founders, priority
feature requests — in exchange for bi-weekly structured feedback calls and right to
use as an anonymised reference.

Do **not** charge design partners.

### Design Partner Qualification Criteria

- 50+ employees
- Technical team capable of self-hosted deployment
- Active automation / integration pain point already identified
- Willing to join 2 feedback calls per month

### What to Extract from Design Partners

- Which enterprise features are blockers vs. nice-to-haves
- What does their procurement / legal process look like?
- What would make them pay, and how much?
- Which integrations are missing?
- Would they allow a public case study (even anonymised)?

### Phase 2 Fast-Fail KPIs (End of Month 6)

| Metric | Minimum (continue) | Strong |
|---|---|---|
| Active design partners | 2 | 5 |
| Partners running production workloads | 1 | 3 |
| Partners willing to pay enterprise license | 1 | 3 |
| Qualified inbound pipeline | 5 | 15+ |
| GitHub stars | 1,000 | 3,000+ |

**If minimum thresholds not reached:** do not build a sales team. Revisit
positioning and ICP before Phase 3.

---

## 5. Phase 3 — First Revenue (Months 6–12)
*Cost: €300–600/month infrastructure + content · Target: €5–15k MRR*

### Pricing Model

**Community Edition:** free, self-hosted, all core features, Polyform Perimeter

**Enterprise License** (license key, self-hosted):

| Tier | Price | Includes |
|---|---|---|
| Starter | €299/month | SSO/SAML, basic RBAC, audit logs, up to 10 users, email support |
| Professional | €799/month | Full RBAC, Git versioning, advanced audit, unlimited users, priority support |
| Enterprise | Custom (€2,000+/month) | Multi-tenant, dedicated support, SLA, custom integrations, on-premise deployment support |

**Founding Partner discount:** 40% off for the first 20 enterprise customers,
locked in for 2 years — rewards early adopters.

**Startup programme:** 60% discount for companies under 20 employees.

### Implemented Enterprise Functionalities & License Protection Audit

Based on a technical audit of the codebase (`internal/enterprise/` and `internal/api/`), the following enterprise features exist, along with their current license protection status:

- **Workflow Limits (Max Runs, Max Workflows)**: ✅ **Protected.** Hard limits are mathematically enforced in `internal/api/handlers.go` based on the signed `LicenseClaims`.
- **Workflow Subprocesses**: ✅ **Protected.** Checked during execution in `internal/temporal/workflow.go` (requires Pro or Enterprise tier).
- **Single Sign-On (SSO) / OIDC**: ⚠️ **Not Protected.** Accessible via `/api/v1/enterprise/oidc` to any user with the `admin` role. Not gated by license tier.
- **Custom Branding**: ⚠️ **Not Protected.** Accessible via `/api/v1/enterprise/branding` to any user with the `admin` role.
- **Environment Secrets Management**: ⚠️ **Not Protected.** Accessible via `/api/v1/enterprise/secrets` to any user with the `admin` role.
- **Audit Logs**: ⚠️ **Not Protected.** Accessible via `/api/v1/enterprise/audit-logs` to any user with the `admin` role.
- **Performance Analytics**: ⚠️ **Not Protected.** Accessible via `/api/v1/enterprise/analytics` to any user with the `admin` role.

> **Crucial Gap Identified:** Currently, only resource usage (runs/workflows) and subprocesses are strictly enforced by the cryptographic license key (`h.getLicenseClaims(ctx)`). The other enterprise features (SSO, Audit Logs, Secrets) are merely protected by standard RBAC (the `admin` role), meaning anyone self-hosting the Community Edition could technically use them. To align with the Pricing Model, `HasFeature` or `Tier` checks must be added to `internal/api/enterprise_handlers.go` before Phase 3.

### Self-Serve Revenue Activation

- Pricing page live with clear tier comparison (currently missing — build before Phase 3)
- Stripe integration for Starter and Professional self-serve
- 14-day trial of enterprise features via temporary license key (no credit card for Starter)
- Enterprise tier: sales-assisted only (form → call → contract)

### Optional: Managed Cloud

Only build if 5+ design partners explicitly say self-hosting is the reason they
won't pay. Host on Hetzner Frankfurt. Do not build speculatively.

### Sales Motion (Founder-Led)

1. Inbound lead arrives
2. Founder responds within 24 hours — short personal video or voice note
3. 30-minute discovery call
4. Customised proposal + 7-day license key for full evaluation
5. Follow up at day 3 and day 7
6. Close or get explicit "not now" with a reason

Target: 1 new enterprise customer per week by month 12.

### Phase 3 Fast-Fail KPIs (End of Month 12)

| Metric | Minimum (continue to Phase 4) | Strong |
|---|---|---|
| MRR | €3,000 | €10,000+ |
| Paying enterprise customers | 3 | 10+ |
| MoM MRR growth | 10% | 20%+ |
| GitHub stars | 2,000 | 5,000+ |
| Active self-hosted deployments (telemetry) | 200 | 800+ |
| Monthly churn | <10% | <3% |
| NPS (design partners) | >40 | >60 |

**If MRR below €3k at month 12 despite consistent marketing:** the market is not
pulling. A sales team will not fix a demand problem. This is the hard gate before
any external fundraising conversation.

---

## 6. Phase 4 — Investor Readiness (Month 12+)
*Only enter this phase if Phase 3 KPIs are met*

### What makes a fundable story

- **Pull metrics:** GitHub stars, community installs, inbound inquiries
- **Revenue quality:** MRR growth rate matters more than absolute MRR
- **Enterprise evidence:** 3+ paying customers with case studies
- **Differentiation:** why this versus n8n, Temporal, Windmill, Power Automate — specific, not generic

### Target Investors

**Primary (growth-stage, AI enterprise thesis, active 2025–2026):**
- Accel (London) — led n8n's Series C; direct comparable investment
- Atomico Growth — dedicated growth vehicle, enterprise data automation thesis
- Balderton Capital — application-layer AI focus

**Secondary (Series A/B, strong enterprise SaaS track record):**
- HV Capital (Munich) — DACH strength, enterprise software through Series D
- Earlybird — backed UiPath; deep automation pattern recognition
- Point Nine (Berlin) — B2B SaaS specialist, co-invests with Accel/Insight

**Do not approach VCs before month 12.** A "not now" with no traction becomes a
permanent perception problem with that firm.

### Fundraising Preparation (Start Month 10)

- [ ] Clean cap table documentation
- [ ] MRR dashboard exportable for due diligence
- [ ] Telemetry data showing install and usage growth
- [ ] 2–3 customer reference contacts willing to speak to investors
- [ ] 12-month financial model with clear use of funds
- [ ] Competitive landscape deck (honest about strengths and weaknesses)

---

## 7. Remaining Pre-Launch Tasks (Priority Order)

These are the gaps between the original plan and the current codebase state,
ordered by impact:

1. **Opt-in telemetry** — anonymous install beacon; without this, Phase 1 KPIs for
   "opt-in installs" cannot be measured
2. **Discord server + README link** — community cannot form without a home
3. **`CONTRIBUTING.md`** — needed before any public GitHub activity
4. **Pricing page** — required before Phase 3 self-serve activation
5. **README comparison table** — differentiates on first impression
6. **Demo video** — highest-leverage content asset for Show HN
7. **One-click deploy template** — reduces friction for the first 100 installs

---

## 8. Cost Summary

| Phase | Duration | Monthly Cost | Cumulative |
|---|---|---|---|
| Pre-launch | 4 weeks | ~€0 | €0 |
| Phase 1: Community | Months 1–3 | €150–300 | €450–900 |
| Phase 2: Design partners | Months 3–6 | €150–300 | €900–1,800 |
| Phase 3: First revenue | Months 6–12 | €300–600 | €2,700–5,400 |
| **Total year 1** | | | **€4,000–8,100** |

No paid advertising. No sales team. No free cloud tier. Every euro builds an asset.

---

## 9. Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| License | Polyform Perimeter 1.0.0 (whole repo) | Allows free commercial internal use; blocks competing products / re-hosting; single license reduces legal ambiguity |
| Repository structure | Monorepo, enterprise in `internal/enterprise/` | Operational simplicity; enterprise code auditable; clean contributor boundary |
| Enterprise gating | License key (product gate, not license gate) | Enterprises can read and audit all code; commercial agreement is about the key |
| Cloud offering | Sandbox on request (48h, email-gated) | Zero upfront cost; no commitment to managed hosting |
| Paid cloud | Only if design partners explicitly request it | Avoid speculative infrastructure |
| Enterprise pricing | Self-serve Starter/Pro, sales-assisted Enterprise | Matches buyer journey |
| VC approach | Not before month 12 / Phase 3 KPIs met | Traction transforms the conversation and the valuation |
| Sales team | Not before €5k+ MRR with clear inbound signal | Sales amplifies demand; it does not create it |
