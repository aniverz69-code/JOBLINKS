-- Curated "remote hiring regions" facet on companies (see the company-remote-regions
-- change). remote_regions records the macro-regions where a company hires remotely,
-- loaded from an external directory by cmd/backfill-remote-regions — a DECLARED
-- company fact, distinct from the job-derived companies.regions facet (the regions
-- our crawled postings sit in). It is NOT maintained by the facet recompute
-- (RefreshCompanyFacets / cmd/recount-companies), which never references this column,
-- so a backfilled value survives every recompute. Values are macro-region codes from
-- enrich.RegionValues; the raw source string is kept under
-- company_info.remote_regions_raw for mapping audit.
--
-- Applied to a fresh volume by initdb after 0004; on an existing prod volume this
-- statement must be run manually BEFORE deploying code that reads the column.
-- Additive with a non-NULL default — every existing company reads as the empty set
-- (no remote-regions loaded) until the backfill annotates it.

ALTER TABLE public.companies
    ADD COLUMN remote_regions text[] DEFAULT '{}'::text[] NOT NULL;
