-- freehire schema — single collapsed init.
--
-- Collapsed from the historical 0001..0043 migration sequence: a schema-only
-- pg_dump of a fully-migrated database. Verified equivalent by a schema
-- round-trip and by a zero-diff `make sqlc` at collapse time. Backfill/transform
-- statements from the old migrations are intentionally dropped — they were
-- no-ops on a fresh (empty) database.
--
-- This file is applied once by Postgres initdb on first volume init and is the
-- schema source for sqlc. Existing databases (prod) already have the full
-- sequence applied and are unaffected; this is the new baseline for fresh envs.

--
-- PostgreSQL database dump
--


-- Dumped from database version 16.14
-- Dumped by pg_dump version 16.14

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: estimate_open_jobs(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.estimate_open_jobs() RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    plan json;
BEGIN
    EXECUTE 'EXPLAIN (FORMAT json) SELECT 1 FROM jobs WHERE closed_at IS NULL'
        INTO plan;
    RETURN (plan -> 0 -> 'Plan' ->> 'Plan Rows')::bigint;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: api_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_keys (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name text NOT NULL,
    token_hash text NOT NULL,
    token_prefix text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone,
    expires_at timestamp with time zone
);


--
-- Name: api_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.api_keys ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.api_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: companies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.companies (
    slug text NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    collections text[] DEFAULT '{}'::text[] NOT NULL,
    job_count integer DEFAULT 0 NOT NULL,
    regions text[] DEFAULT '{}'::text[] NOT NULL,
    countries text[] DEFAULT '{}'::text[] NOT NULL,
    domains text[] DEFAULT '{}'::text[] NOT NULL,
    company_types text[] DEFAULT '{}'::text[] NOT NULL,
    company_sizes text[] DEFAULT '{}'::text[] NOT NULL,
    industries text[] DEFAULT '{}'::text[] NOT NULL,
    year_founded integer,
    employee_count integer,
    hq_country text,
    organization_type text,
    tagline text,
    company_info jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_reference boolean DEFAULT false NOT NULL,
    company_info_at timestamp with time zone
);


--
-- Name: enrichment_outbox; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.enrichment_outbox (
    id bigint NOT NULL,
    job_id bigint NOT NULL,
    target_version integer NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    claimed_at timestamp with time zone,
    failed_at timestamp with time zone,
    last_error text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: enrichment_outbox_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.enrichment_outbox ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.enrichment_outbox_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: job_reports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.job_reports (
    id bigint NOT NULL,
    reported_by bigint NOT NULL,
    job_id bigint NOT NULL,
    reason text NOT NULL,
    details text NOT NULL,
    contact_telegram text DEFAULT ''::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    review_reason text DEFAULT ''::text NOT NULL,
    reviewed_by bigint,
    reviewed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT job_reports_reason_check CHECK ((reason = ANY (ARRAY['no_response'::text, 'not_relevant'::text, 'spam'::text, 'fraud'::text, 'other'::text]))),
    CONSTRAINT job_reports_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'resolved'::text, 'dismissed'::text])))
);


--
-- Name: job_reports_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.job_reports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_reports_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.job_reports_id_seq OWNED BY public.job_reports.id;


--
-- Name: job_submissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.job_submissions (
    id bigint NOT NULL,
    submitted_by bigint NOT NULL,
    url text NOT NULL,
    source text DEFAULT ''::text NOT NULL,
    title text NOT NULL,
    company text NOT NULL,
    location text DEFAULT ''::text NOT NULL,
    remote boolean DEFAULT false NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    posted_at timestamp with time zone,
    status text DEFAULT 'pending'::text NOT NULL,
    review_reason text DEFAULT ''::text NOT NULL,
    reviewed_by bigint,
    reviewed_at timestamp with time zone,
    job_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT job_submissions_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text])))
);


--
-- Name: job_submissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.job_submissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_submissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.job_submissions_id_seq OWNED BY public.job_submissions.id;


--
-- Name: jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.jobs (
    id bigint NOT NULL,
    source text NOT NULL,
    external_id text NOT NULL,
    url text NOT NULL,
    title text NOT NULL,
    company text DEFAULT ''::text NOT NULL,
    location text DEFAULT ''::text NOT NULL,
    remote boolean DEFAULT true NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    posted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    company_slug text DEFAULT ''::text NOT NULL,
    enrichment jsonb DEFAULT '{}'::jsonb NOT NULL,
    enriched_at timestamp with time zone,
    enrichment_version integer DEFAULT 0 NOT NULL,
    public_slug text NOT NULL,
    last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    closed_at timestamp with time zone,
    countries text[] DEFAULT '{}'::text[] NOT NULL,
    regions text[] DEFAULT '{}'::text[] NOT NULL,
    work_mode text DEFAULT ''::text NOT NULL,
    liveness_strikes integer DEFAULT 0 NOT NULL,
    skills text[] DEFAULT '{}'::text[] NOT NULL,
    seniority text DEFAULT ''::text NOT NULL,
    category text DEFAULT ''::text NOT NULL,
    created_by bigint,
    updated_by bigint,
    posting_language text DEFAULT ''::text NOT NULL,
    employment_type text DEFAULT ''::text NOT NULL,
    education_level text DEFAULT ''::text NOT NULL,
    experience_years_min integer,
    collections text[] DEFAULT '{}'::text[] NOT NULL,
    content_hash text,
    english_level text DEFAULT ''::text NOT NULL,
    cities text[] DEFAULT '{}'::text[] NOT NULL,
    view_count integer DEFAULT 0 NOT NULL,
    applied_count integer DEFAULT 0 NOT NULL
);


--
-- Name: jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.jobs ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: saved_searches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.saved_searches (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name text NOT NULL,
    query text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    public_slug text,
    author_label text,
    CONSTRAINT saved_searches_name_check CHECK (((length(TRIM(BOTH FROM name)) >= 1) AND (length(TRIM(BOTH FROM name)) <= 100)))
);


--
-- Name: saved_searches_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.saved_searches ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.saved_searches_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: subscription_matches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscription_matches (
    subscription_id bigint NOT NULL,
    job_id bigint NOT NULL,
    matched_at timestamp with time zone DEFAULT now() NOT NULL,
    notified_at timestamp with time zone,
    claimed_at timestamp with time zone,
    attempts integer DEFAULT 0 NOT NULL,
    failed_at timestamp with time zone,
    last_error text DEFAULT ''::text NOT NULL
);


--
-- Name: subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscriptions (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    saved_search_id bigint NOT NULL,
    channel text DEFAULT 'telegram'::text NOT NULL,
    destination text,
    active boolean DEFAULT true NOT NULL,
    start_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: subscriptions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.subscriptions ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.subscriptions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: telegram_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.telegram_links (
    user_id bigint NOT NULL,
    chat_id bigint NOT NULL,
    linked_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: telegram_posts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.telegram_posts (
    channel text NOT NULL,
    msg_id bigint NOT NULL,
    text text NOT NULL,
    posted_at timestamp with time zone NOT NULL,
    fetched_at timestamp with time zone DEFAULT now() NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    claimed_at timestamp with time zone,
    failed_at timestamp with time zone,
    last_error text DEFAULT ''::text NOT NULL,
    extracted_at timestamp with time zone,
    links jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: user_identities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_identities (
    provider text NOT NULL,
    provider_user_id text NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_jobs (
    user_id bigint NOT NULL,
    job_id bigint NOT NULL,
    viewed_at timestamp with time zone DEFAULT now() NOT NULL,
    applied_at timestamp with time zone,
    saved_at timestamp with time zone,
    stage text,
    notes text,
    dismissed_at timestamp with time zone
);


--
-- Name: user_profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_profiles (
    user_id bigint NOT NULL,
    skills text[] NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    specializations text[] NOT NULL,
    CONSTRAINT search_profiles_skills_check CHECK ((cardinality(skills) > 0)),
    CONSTRAINT user_profiles_specializations_card_chk CHECK (((cardinality(specializations) >= 1) AND (cardinality(specializations) <= 5)))
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    email text NOT NULL,
    password_hash text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    role text DEFAULT 'user'::text NOT NULL,
    resume_object_key text,
    resume_uploaded_at timestamp with time zone,
    resume_ats_analysis jsonb,
    CONSTRAINT users_role_check CHECK ((role = ANY (ARRAY['user'::text, 'moderator'::text, 'admin'::text])))
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.users ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: job_reports id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_reports ALTER COLUMN id SET DEFAULT nextval('public.job_reports_id_seq'::regclass);


--
-- Name: job_submissions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_submissions ALTER COLUMN id SET DEFAULT nextval('public.job_submissions_id_seq'::regclass);


--
-- Name: api_keys api_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);


--
-- Name: companies companies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.companies
    ADD CONSTRAINT companies_pkey PRIMARY KEY (slug);


--
-- Name: enrichment_outbox enrichment_outbox_job_id_target_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.enrichment_outbox
    ADD CONSTRAINT enrichment_outbox_job_id_target_version_key UNIQUE (job_id, target_version);


--
-- Name: enrichment_outbox enrichment_outbox_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.enrichment_outbox
    ADD CONSTRAINT enrichment_outbox_pkey PRIMARY KEY (id);


--
-- Name: job_reports job_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_reports
    ADD CONSTRAINT job_reports_pkey PRIMARY KEY (id);


--
-- Name: job_submissions job_submissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_submissions
    ADD CONSTRAINT job_submissions_pkey PRIMARY KEY (id);


--
-- Name: jobs jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);


--
-- Name: jobs jobs_public_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_public_slug_key UNIQUE (public_slug);


--
-- Name: jobs jobs_source_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_source_external_id_key UNIQUE (source, external_id);


--
-- Name: saved_searches saved_searches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_pkey PRIMARY KEY (id);


--
-- Name: saved_searches saved_searches_user_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_user_id_name_key UNIQUE (user_id, name);


--
-- Name: user_profiles search_profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_profiles
    ADD CONSTRAINT search_profiles_pkey PRIMARY KEY (user_id);


--
-- Name: subscription_matches subscription_matches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscription_matches
    ADD CONSTRAINT subscription_matches_pkey PRIMARY KEY (subscription_id, job_id);


--
-- Name: subscriptions subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_pkey PRIMARY KEY (id);


--
-- Name: subscriptions subscriptions_saved_search_id_channel_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_saved_search_id_channel_key UNIQUE (saved_search_id, channel);


--
-- Name: telegram_links telegram_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.telegram_links
    ADD CONSTRAINT telegram_links_pkey PRIMARY KEY (user_id);


--
-- Name: telegram_posts telegram_posts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.telegram_posts
    ADD CONSTRAINT telegram_posts_pkey PRIMARY KEY (channel, msg_id);


--
-- Name: user_identities user_identities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_pkey PRIMARY KEY (provider, provider_user_id);


--
-- Name: user_jobs user_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_jobs
    ADD CONSTRAINT user_jobs_pkey PRIMARY KEY (user_id, job_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: api_keys_token_hash_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX api_keys_token_hash_idx ON public.api_keys USING btree (token_hash);


--
-- Name: api_keys_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_keys_user_id_idx ON public.api_keys USING btree (user_id);


--
-- Name: companies_company_sizes_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX companies_company_sizes_idx ON public.companies USING gin (company_sizes);


--
-- Name: companies_company_types_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX companies_company_types_idx ON public.companies USING gin (company_types);


--
-- Name: companies_countries_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX companies_countries_idx ON public.companies USING gin (countries);


--
-- Name: companies_domains_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX companies_domains_idx ON public.companies USING gin (domains);


--
-- Name: companies_industries_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX companies_industries_idx ON public.companies USING gin (industries);


--
-- Name: companies_job_count_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX companies_job_count_idx ON public.companies USING btree (job_count DESC, name);


--
-- Name: companies_regions_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX companies_regions_idx ON public.companies USING gin (regions);


--
-- Name: enrichment_outbox_claimable_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX enrichment_outbox_claimable_idx ON public.enrichment_outbox USING btree (id) WHERE (failed_at IS NULL);


--
-- Name: job_reports_open_user_job_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX job_reports_open_user_job_key ON public.job_reports USING btree (reported_by, job_id) WHERE (status = 'pending'::text);


--
-- Name: job_reports_pending_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX job_reports_pending_idx ON public.job_reports USING btree (created_at DESC) WHERE (status = 'pending'::text);


--
-- Name: job_submissions_by_user_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX job_submissions_by_user_idx ON public.job_submissions USING btree (submitted_by, created_at DESC);


--
-- Name: job_submissions_pending_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX job_submissions_pending_idx ON public.job_submissions USING btree (created_at DESC) WHERE (status = 'pending'::text);


--
-- Name: job_submissions_pending_url_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX job_submissions_pending_url_key ON public.job_submissions USING btree (lower(url)) WHERE (status = 'pending'::text);


--
-- Name: jobs_company_slug_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_company_slug_idx ON public.jobs USING btree (company_slug, posted_at DESC NULLS LAST, id DESC);


--
-- Name: jobs_open_company_created_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_open_company_created_at_id_idx ON public.jobs USING btree (company_slug, created_at DESC, id DESC) WHERE (closed_at IS NULL);


--
-- Name: jobs_open_created_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_open_created_idx ON public.jobs USING btree (created_at DESC, id DESC) WHERE (closed_at IS NULL);


--
-- Name: jobs_open_enrich_freshness_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_open_enrich_freshness_idx ON public.jobs USING btree (COALESCE(posted_at, created_at) DESC, id DESC) WHERE (closed_at IS NULL);


--
-- Name: jobs_open_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_open_id_idx ON public.jobs USING btree (id) WHERE (closed_at IS NULL);


--
-- Name: jobs_posted_at_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_posted_at_id_idx ON public.jobs USING btree (posted_at DESC NULLS LAST, id DESC);


--
-- Name: jobs_source_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX jobs_source_idx ON public.jobs USING btree (source);


--
-- Name: saved_searches_public_slug_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX saved_searches_public_slug_idx ON public.saved_searches USING btree (public_slug);


--
-- Name: saved_searches_user_updated_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX saved_searches_user_updated_idx ON public.saved_searches USING btree (user_id, updated_at DESC);


--
-- Name: subscription_matches_pending_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX subscription_matches_pending_idx ON public.subscription_matches USING btree (subscription_id) WHERE ((notified_at IS NULL) AND (failed_at IS NULL));


--
-- Name: subscriptions_user_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX subscriptions_user_idx ON public.subscriptions USING btree (user_id);


--
-- Name: telegram_posts_claimable_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX telegram_posts_claimable_idx ON public.telegram_posts USING btree (posted_at) WHERE ((extracted_at IS NULL) AND (failed_at IS NULL));


--
-- Name: user_identities_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_identities_user_id_idx ON public.user_identities USING btree (user_id);


--
-- Name: users_email_lower_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX users_email_lower_idx ON public.users USING btree (lower(email));


--
-- Name: api_keys api_keys_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: enrichment_outbox enrichment_outbox_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.enrichment_outbox
    ADD CONSTRAINT enrichment_outbox_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id) ON DELETE CASCADE;


--
-- Name: job_reports job_reports_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_reports
    ADD CONSTRAINT job_reports_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id) ON DELETE CASCADE;


--
-- Name: job_reports job_reports_reported_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_reports
    ADD CONSTRAINT job_reports_reported_by_fkey FOREIGN KEY (reported_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: job_reports job_reports_reviewed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_reports
    ADD CONSTRAINT job_reports_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: job_submissions job_submissions_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_submissions
    ADD CONSTRAINT job_submissions_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id) ON DELETE SET NULL;


--
-- Name: job_submissions job_submissions_reviewed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_submissions
    ADD CONSTRAINT job_submissions_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: job_submissions job_submissions_submitted_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.job_submissions
    ADD CONSTRAINT job_submissions_submitted_by_fkey FOREIGN KEY (submitted_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: jobs jobs_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: jobs jobs_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: saved_searches saved_searches_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.saved_searches
    ADD CONSTRAINT saved_searches_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_profiles search_profiles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_profiles
    ADD CONSTRAINT search_profiles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: subscription_matches subscription_matches_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscription_matches
    ADD CONSTRAINT subscription_matches_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id) ON DELETE CASCADE;


--
-- Name: subscription_matches subscription_matches_subscription_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscription_matches
    ADD CONSTRAINT subscription_matches_subscription_id_fkey FOREIGN KEY (subscription_id) REFERENCES public.subscriptions(id) ON DELETE CASCADE;


--
-- Name: subscriptions subscriptions_saved_search_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_saved_search_id_fkey FOREIGN KEY (saved_search_id) REFERENCES public.saved_searches(id) ON DELETE CASCADE;


--
-- Name: subscriptions subscriptions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: telegram_links telegram_links_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.telegram_links
    ADD CONSTRAINT telegram_links_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_identities user_identities_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_identities
    ADD CONSTRAINT user_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_jobs user_jobs_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_jobs
    ADD CONSTRAINT user_jobs_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id) ON DELETE CASCADE;


--
-- Name: user_jobs user_jobs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_jobs
    ADD CONSTRAINT user_jobs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--


