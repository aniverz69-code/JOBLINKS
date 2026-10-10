-- More curated YC directory facets on companies (see the yc-max-enrichment change).
-- yc_stage ('Early'/'Growth') and yc_flags ('top_company','hiring') are loaded from
-- the yc-oss directory by cmd/import-yc — curated company facts, NOT job-derived, and
-- NOT maintained by RefreshCompanyFacets / cmd/recount-companies (which never
-- references them). Modelled as text[] so they filter through the same array-overlap
-- facet machinery as yc_batch/yc_status.
--
-- Applied to a fresh volume by initdb after 0007; on an existing prod volume these
-- statements must be run manually BEFORE deploying code that reads the columns.
-- Additive with a non-NULL default — empty until the importer annotates a company.

ALTER TABLE public.companies
    ADD COLUMN yc_stage text[] DEFAULT '{}'::text[] NOT NULL,
    ADD COLUMN yc_flags text[] DEFAULT '{}'::text[] NOT NULL;
