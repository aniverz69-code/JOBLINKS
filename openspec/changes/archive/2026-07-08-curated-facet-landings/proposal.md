## Why

freehire ships curated SEO landing pages for filter collections at
`/collections/[slug]`, but the set covers only two axes (a few remote regions and
~35 languages/frameworks), leaving strong "<X> jobs" search terms across other
facets — tech categories, seniority, infra tools, named roles — uncaptured. The
`job-collections` spec is also stale: it still says there is no per-collection
page, which the shipped landing pages contradict.

## What Changes

- Extend the frontend-only `FILTER_COLLECTIONS` registry with ~33 more
  hand-verified filter collections across four new facet axes: tech categories
  (`category=`), seniority (`seniority=`), infra skills (`skills=`), and named tech
  roles (`role=`).
- Each new value is curated (not programmatic) and confirmed to have a live
  job-count ≥ 300 before shipping, keeping the thin-content risk off the table.
- Reconcile the `job-collections` spec with reality: a filter collection renders at
  a dedicated `/collections/[slug]` SEO landing page (self-canonical, breadcrumb
  JSON-LD, scoped feed), replacing the stale "no per-collection page" requirement.
- Add a unit test asserting every `FILTER_COLLECTIONS` slug is unique and every
  entry has non-empty `params`.
- No backend, routing, sitemap, or page-template changes — the landing route,
  `excludeFacets` behaviour, and `collectionPaths()` sitemap inclusion already
  exist.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `job-collections`: a filter collection renders at a dedicated `/collections/[slug]`
  SEO landing page (superseding "no per-collection page"), and its `params` may pin
  any `/jobs` facet axis (categories, seniority, roles, skills), not only
  work-mode/region.

## Impact

- `web/src/lib/collections.ts` — new `FILTER_COLLECTIONS` entries (data only).
- `web/src/lib/collections.test.ts` (new) — slug-uniqueness + non-empty-params guard.
- `openspec/specs/job-collections/spec.md` — delta reconciling the landing-page
  requirement.
- No API, DB, or worker changes. New pages ride the existing `/collections/[slug]`
  route and `collectionPaths()` sitemap.
