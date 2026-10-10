## 1. Schema

- [x] 1.1 Migration `migrations/0008_company_yc_stage_flags.sql` adding
      `companies.yc_stage text[]` + `yc_flags text[]` (NOT NULL DEFAULT '{}').
- [x] 1.2 Regenerate `internal/db`; confirm `db.Company` gains `YcStage`/`YcFlags`.

## 2. ycdir mapping

- [x] 2.1 RED: extend `ycdir` tests — richer industries union
      (industry+industries+subindustry+tags deduped), `Stage` passthrough,
      `FormerSlugs` (normalized former_names), `Flags` ({top_company,hiring}).
- [x] 2.2 GREEN: add `Industries`/`Subindustry`/`FormerNames`/`Stage`/`TopCompany`/
      `IsHiring` to the entry + map them; tests green.
- [x] 2.3 REFACTOR + simplify.

## 3. Persistence

- [x] 3.1 Add `yc_stage`/`yc_flags` to `UpsertYCCompany` (INSERT + ON CONFLICT);
      regenerate sqlc.
- [x] 3.2 Integration test: the two new facets are set on upsert and left untouched
      by `RefreshCompanyFacets`.

## 4. Importer former-name matching

- [x] 4.1 RED: loader test — an entry whose current slug is absent but a former slug
      exists updates the existing company (counted matched), not inserted.
- [x] 4.2 GREEN: resolve target slug via `CompanyExists` over
      `[slug(name), former slugs…]`; pass yc_stage/yc_flags through recordToParams.
- [x] 4.3 REFACTOR + simplify.

## 5. Company list facets (API)

- [x] 5.1 Add `yc_stage`/`yc_flags` params to `ListCompanies`/`CountCompanies`;
      regenerate. Wire through the handler.
- [x] 5.2 Handler test: `?yc_stage=Growth&yc_flags=top_company` filters + total.

## 6. Web

- [x] 6.1 Add `yc_stage` (pills Early/Growth) + `yc_flags` (pills: Top company,
      Hiring) facets to `COMPANY_FACETS`; model round-trip test.
- [x] 6.2 Company detail badges: stage / "YC Top Company" / "Hiring" from the
      returned facets/company_info (reuse `CompanyFacts`/`CompanyHeader`).
- [x] 6.3 Verify (svelte-check + vitest).

## 7. Docs + finish + deploy

- [x] 7.1 AGENT.md: former-name matching, richer industries, yc_stage/yc_flags.
- [x] 7.2 `go build/vet/test` + integration green.
- [ ] 7.3 Migration on prod, deploy, re-run `cmd/import-yc`, verify facets/badges live.
