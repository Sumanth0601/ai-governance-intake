# AI Governance Intake

Built this to understand the core intake loop that AI governance products are built around — specifically the two things that matter: catching duplicate AI initiatives before they get funded twice, and auto-scoring proposals against compliance frameworks so teams don't have to do it manually.

**The problem it solves:** When a company wants to launch a new AI project, someone has to review it for compliance risk (data privacy, bias, human oversight, regulatory classification). Today that's manual, slow, and inconsistent. This automates it.

---

## How it works

1. Someone submits an AI project proposal via the form
2. The text gets embedded and compared against all existing proposals — if it's too similar to something already in the system, it's flagged as a duplicate and stopped there
3. If it's new, an LLM scores it against ISO 42001 and NIST AI RMF controls and returns a structured risk scorecard
4. Everything is saved and visible on the dashboard

---

## Example

A company submits a proposal to build an AI resume screener. Here's what comes back:

```json
{
  "risk_score": "high",
  "scorecard": {
    "overall_risk": "high",
    "findings": [
      {
        "control": "ISO 42001 § 8.5 — Human Oversight",
        "severity": "high",
        "issue": "Automatically rejecting 60% of applicants with no human review is a significant liability.",
        "recommendation": "Require a recruiter to spot-check a sample of auto-rejected candidates each week."
      },
      {
        "control": "NIST AI RMF — Bias & Fairness",
        "severity": "critical",
        "issue": "Resume screening models are known to inherit hiring biases — no bias testing is mentioned.",
        "recommendation": "Run bias audits across gender, age, and ethnicity before deployment."
      }
    ],
    "summary": "Using AI to filter job applicants is high-risk by default — it directly affects people's livelihoods and is regulated in several countries. Main gaps: no human oversight, no bias testing, unclear data retention."
  }
}
```

If the same proposal is submitted again later:

```json
{
  "status": "duplicate",
  "duplicate_of": "c0f88038-74c9-4120-83c7-9ddb24660a15",
  "similarity": 0.97
}
```

---

## Stack

- **Go** — `chi` router, `pgx/v5`
- **Supabase** — Postgres + `pgvector` for similarity search
- **OpenRouter** — embeddings + LLM scoring (both free tier)
- **Frontend** — single HTML file, vanilla JS, no framework, no build step

---

## Running it

```bash
cp .env.example .env
# fill in DATABASE_URL and OPENROUTER_API_KEY

go run ./cmd/server
# opens on http://localhost:8080
# migrations run automatically on startup
```

Requires a Supabase project with `pgvector` enabled (`CREATE EXTENSION IF NOT EXISTS vector;`).

