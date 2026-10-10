## Context

`cmd/import-yc` + `internal/ycdir` already load the yc-oss directory into the
company-info columns and the curated `yc_batch`/`yc_status` facets. This extends
that same pipeline to the remaining high-value fields, reusing every pattern
already in place (curated `text[]` facets filtered by `&&`, recompute-exempt,
FilterModal pills).

## Goals / Non-Goals

**Goals:** former-name matching, richer industries, `yc_stage`/`yc_flags` facets +
UI, company-page badges.

**Non-Goals:** wiring yc-oss logos (logo.dev already serves logos by name); the
per-company yc-oss API (founders/jobs) — a separate, larger source project.

## Decisions

**Former-name matching in the importer, not a rename.** `ycdir.Map` exposes each
former name's normalized slug. `cmd/import-yc` resolves an entry's target as the
first of `[slug(name), slug(former₁), …]` that `CompanyExists`, else `slug(name)`
(insert). The upsert never overwrites `name` on conflict, so matching a former slug
enriches the existing row without renaming it. This cuts duplicate reference rows
(48% of entries carry former names).

**`yc_flags` as a single `text[]`.** `top_company`/`isHiring` are booleans, but
modelling them as membership in one `yc_flags` array (`{top_company, hiring}`)
reuses the `&&` facet machinery with no boolean-filter special case — the same
reason `yc_batch` is `text[]`. `yc_stage` is likewise `text[]` (Early/Growth).

**Richer industries union.** The mapping unions `industry` + `industries[]` +
`subindustry` + `tags`, de-duplicated (order-preserving), replacing the prior
`industry` + `tags`.

**Badges reuse existing company components.** Stage / "YC Top Company" / "Hiring"
render on the company detail view from the stored facets/company_info, alongside
the existing `CompanyFacts`/`CompanyHeader` — no new data plumbing beyond what the
detail response already returns.

## Risks / Trade-offs

- **Former-name collisions** (a former name normalizing onto an unrelated existing
  company). → Low risk (YC former names are specific); the importer only enriches
  company-info/facets, never renames or merges, so a wrong match is a bounded,
  reversible data blemish fixed on the next run.
- **Re-running shifts counts** (former-name matching turns some prior inserts into
  updates). → Expected and desirable; the run logs matched vs inserted.

## Migration Plan

1. Migration (`yc_stage`, `yc_flags`) + code; regenerate sqlc.
2. Deploy; re-run `cmd/import-yc` (former-name matching + new facets populate).
3. Rollback: additive columns, additive facets — revert binary, columns idle.

## Open Questions

- None. Logos and the per-company API are explicit non-goals.
