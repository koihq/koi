-- Baseline: the schema of the first release, v0.1.0. Never edit a released migration; add a new
-- one with `make migration name=...`. River's tables are managed by rivermigrate, not here.

-- +goose Up
CREATE TABLE billing_change_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    organization_id uuid NOT NULL,
    idempotency_key character varying(255) NOT NULL,
    subscription_id text NOT NULL,
    quantity bigint NOT NULL,
    result jsonb,
    error_code text DEFAULT ''::text NOT NULL
);

CREATE TABLE billing_checkout_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    organization_id uuid NOT NULL,
    session_id text NOT NULL,
    url text DEFAULT ''::text NOT NULL,
    expires_at timestamp with time zone,
    customer_id text NOT NULL,
    price_id text NOT NULL,
    quantity bigint NOT NULL,
    trial boolean DEFAULT false NOT NULL,
    completed boolean DEFAULT false NOT NULL,
    retired boolean DEFAULT false NOT NULL,
    stripe_subscription_id text
);

CREATE TABLE billing_subscriptions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    organization_id uuid NOT NULL,
    stripe_subscription_id text NOT NULL,
    stripe_price_id text NOT NULL,
    quantity bigint NOT NULL,
    status text NOT NULL,
    current_period_end timestamp with time zone,
    trial_end timestamp with time zone,
    cancel_at_period_end boolean DEFAULT false NOT NULL,
    pending_quantity bigint,
    stripe_schedule_id text,
    payment_url text DEFAULT ''::text NOT NULL,
    pending_upgrade_quantity bigint,
    pending_upgrade_expires_at timestamp with time zone
);

CREATE TABLE brands (
    tracking_confirmed_at timestamp with time zone,
    visibility_started_at timestamp with time zone,
    next_visibility_at timestamp with time zone,
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    organization_id uuid NOT NULL,
    name text NOT NULL,
    domain text,
    github_repo text,
    country character varying(2) NOT NULL,
    description text
);

CREATE TABLE competitor_observations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    brand_id uuid NOT NULL,
    key text NOT NULL,
    count bigint NOT NULL,
    dismissed boolean DEFAULT false NOT NULL,
    identity jsonb NOT NULL
);

CREATE TABLE competitors (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    brand_id uuid NOT NULL,
    name text NOT NULL,
    domain text NOT NULL,
    github_repo text DEFAULT ''::text NOT NULL,
    aliases jsonb DEFAULT '[]'::jsonb NOT NULL,
    additional_urls jsonb DEFAULT '[]'::jsonb NOT NULL,
    source character varying(16) NOT NULL,
    confirmed boolean DEFAULT false NOT NULL
);

CREATE TABLE credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    name text NOT NULL,
    token_hash text NOT NULL,
    kind text NOT NULL,
    client_id text,
    connection_id uuid,
    audience text,
    scope text,
    expires_at timestamp with time zone,
    used_at timestamp with time zone
);

CREATE TABLE generation_runs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    brand_id uuid NOT NULL,
    state character varying(16) NOT NULL,
    river_job_id bigint,
    idempotency_key character varying(255),
    attempt bigint DEFAULT 0 NOT NULL,
    prompt_count bigint DEFAULT 0 NOT NULL,
    competitor_count bigint DEFAULT 0 NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    next_retry_at timestamp with time zone,
    error_code text,
    model_slug text NOT NULL,
    brand_snapshot jsonb NOT NULL,
    start_tracking boolean DEFAULT false NOT NULL,
    summary jsonb
);

CREATE TABLE memberships (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    role text NOT NULL
);

CREATE TABLE oauth_clients (
    id text NOT NULL,
    redirect_uris text NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone NOT NULL
);

CREATE TABLE oauth_codes (
    hash text NOT NULL,
    user_id uuid NOT NULL,
    client_id text NOT NULL,
    client_name text NOT NULL,
    redirect_uri text NOT NULL,
    challenge text NOT NULL,
    scope text NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE organizations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name text NOT NULL,
    stripe_customer_id text,
    trial_used_at timestamp with time zone
);

CREATE TABLE password_reset_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE prompts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    brand_id uuid NOT NULL,
    text text NOT NULL,
    source character varying(16) NOT NULL,
    confirmed boolean DEFAULT false NOT NULL,
    inactive boolean DEFAULT false NOT NULL
);

CREATE TABLE sessions (
    token text NOT NULL,
    data bytea NOT NULL,
    expiry timestamp with time zone NOT NULL
);

CREATE TABLE stripe_webhook_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    stripe_event_id text NOT NULL
);

CREATE TABLE users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL
);

CREATE TABLE visibility_results (
    checked_at timestamp with time zone,
    cloro_task_id text,
    submitted_at timestamp with time zone,
    sync_attempts bigint DEFAULT 0 NOT NULL,
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    brand_id uuid NOT NULL,
    run_created_at timestamp with time zone NOT NULL,
    run_id uuid NOT NULL,
    prompt_id uuid NOT NULL,
    prompt_text text NOT NULL,
    prompt_position bigint NOT NULL,
    engine character varying(32) NOT NULL,
    collector character varying(16) NOT NULL,
    fetched_at timestamp with time zone,
    outcome character varying(16),
    raw_answer text DEFAULT ''::text NOT NULL,
    answer_markdown text DEFAULT ''::text NOT NULL,
    named_entities jsonb DEFAULT '[]'::jsonb NOT NULL,
    observations jsonb DEFAULT '[]'::jsonb NOT NULL,
    sources jsonb NOT NULL,
    error_code text,
    provider_credits bigint,
    mention_verdict character varying(16),
    mention_cost_usd numeric,
    CONSTRAINT chk_visibility_mention_verdict CHECK (mention_verdict IS NULL OR mention_verdict IN ('mention','no_info','other_company','other_meaning')),
    CONSTRAINT chk_visibility_outcome CHECK (outcome IS NULL OR outcome IN ('cited','mentioned','miss','error'))
);

CREATE TABLE visibility_runs (
    scheduled_for timestamp with time zone,
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    brand_id uuid NOT NULL,
    fast_start boolean DEFAULT false NOT NULL,
    brand_snapshot jsonb NOT NULL,
    engines jsonb NOT NULL,
    state character varying(16) NOT NULL,
    total_results bigint NOT NULL,
    river_job_id bigint,
    attempt bigint DEFAULT 0 NOT NULL,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    error_code text,
    CONSTRAINT chk_visibility_state CHECK (state IN ('queued','running','succeeded','failed'))
);

ALTER TABLE ONLY billing_change_requests
    ADD CONSTRAINT billing_change_requests_pkey PRIMARY KEY (id);

ALTER TABLE ONLY billing_checkout_sessions
    ADD CONSTRAINT billing_checkout_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY billing_subscriptions
    ADD CONSTRAINT billing_subscriptions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY brands
    ADD CONSTRAINT brands_pkey PRIMARY KEY (id);

ALTER TABLE ONLY competitor_observations
    ADD CONSTRAINT competitor_observations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY competitors
    ADD CONSTRAINT competitors_pkey PRIMARY KEY (id);

ALTER TABLE ONLY credentials
    ADD CONSTRAINT credentials_pkey PRIMARY KEY (id);

ALTER TABLE ONLY generation_runs
    ADD CONSTRAINT generation_runs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY memberships
    ADD CONSTRAINT memberships_pkey PRIMARY KEY (id);

ALTER TABLE ONLY oauth_clients
    ADD CONSTRAINT oauth_clients_pkey PRIMARY KEY (id);

ALTER TABLE ONLY oauth_codes
    ADD CONSTRAINT oauth_codes_pkey PRIMARY KEY (hash);

ALTER TABLE ONLY organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);

ALTER TABLE ONLY prompts
    ADD CONSTRAINT prompts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (token);

ALTER TABLE ONLY stripe_webhook_events
    ADD CONSTRAINT stripe_webhook_events_pkey PRIMARY KEY (id);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE ONLY visibility_results
    ADD CONSTRAINT visibility_results_pkey PRIMARY KEY (id);

ALTER TABLE ONLY visibility_runs
    ADD CONSTRAINT visibility_runs_pkey PRIMARY KEY (id);

CREATE UNIQUE INDEX idx_billing_change_key ON billing_change_requests USING btree (organization_id, idempotency_key);

CREATE INDEX idx_billing_checkout_sessions_organization_id ON billing_checkout_sessions USING btree (organization_id);

CREATE UNIQUE INDEX idx_billing_checkout_sessions_session_id ON billing_checkout_sessions USING btree (session_id);

CREATE UNIQUE INDEX idx_billing_checkout_sessions_stripe_subscription_id ON billing_checkout_sessions USING btree (stripe_subscription_id);

CREATE UNIQUE INDEX idx_billing_subscriptions_organization_id ON billing_subscriptions USING btree (organization_id);

CREATE UNIQUE INDEX idx_billing_subscriptions_stripe_subscription_id ON billing_subscriptions USING btree (stripe_subscription_id);

CREATE INDEX idx_brands_organization_id ON brands USING btree (organization_id);

CREATE UNIQUE INDEX idx_competitor_observations_key ON competitor_observations USING btree (brand_id, key);

CREATE UNIQUE INDEX idx_competitors_brand_domain ON competitors USING btree (brand_id, domain) WHERE (domain <> ''::text);

CREATE UNIQUE INDEX idx_competitors_brand_github_repo ON competitors USING btree (brand_id, github_repo) WHERE (github_repo <> ''::text);

CREATE INDEX idx_credentials_connection_id ON credentials USING btree (connection_id);

CREATE UNIQUE INDEX idx_credentials_token_hash ON credentials USING btree (token_hash);

CREATE INDEX idx_credentials_user_id ON credentials USING btree (user_id);

CREATE UNIQUE INDEX idx_generation_runs_active ON generation_runs USING btree (brand_id) WHERE (((state)::text = 'queued'::text) OR ((state)::text = 'running'::text));

CREATE UNIQUE INDEX idx_generation_runs_key ON generation_runs USING btree (brand_id, idempotency_key) WHERE (idempotency_key IS NOT NULL);

CREATE INDEX idx_generation_runs_recent ON generation_runs USING btree (brand_id, created_at DESC, id DESC);

CREATE INDEX idx_memberships_organization_id ON memberships USING btree (organization_id);

CREATE UNIQUE INDEX idx_memberships_user_id ON memberships USING btree (user_id);

CREATE UNIQUE INDEX idx_organizations_stripe_customer_id ON organizations USING btree (stripe_customer_id);

CREATE UNIQUE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens USING btree (token_hash);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens USING btree (user_id);

CREATE INDEX idx_prompts_brand_id ON prompts USING btree (brand_id);

CREATE UNIQUE INDEX idx_prompts_brand_text ON prompts USING btree (brand_id, md5(lower(text)));

CREATE INDEX idx_sessions_expiry ON sessions USING btree (expiry);

CREATE UNIQUE INDEX idx_stripe_webhook_events_stripe_event_id ON stripe_webhook_events USING btree (stripe_event_id);

CREATE UNIQUE INDEX idx_users_email ON users USING btree (email);

CREATE UNIQUE INDEX idx_visibility_results_cell ON visibility_results USING btree (run_id, prompt_id, engine);

CREATE INDEX idx_visibility_results_order ON visibility_results USING btree (run_id, prompt_position, engine);

CREATE INDEX idx_visibility_results_prompt_window ON visibility_results USING btree (brand_id, prompt_id, run_created_at);

CREATE INDEX idx_visibility_results_window ON visibility_results USING btree (brand_id, run_created_at);

CREATE UNIQUE INDEX idx_visibility_runs_active ON visibility_runs USING btree (brand_id) WHERE (((state)::text = 'queued'::text) OR ((state)::text = 'running'::text));

CREATE INDEX idx_visibility_runs_recent ON visibility_runs USING btree (brand_id, created_at DESC, id DESC);

CREATE UNIQUE INDEX idx_visibility_runs_slot ON visibility_runs USING btree (brand_id, scheduled_for) WHERE (scheduled_for IS NOT NULL);

ALTER TABLE ONLY billing_change_requests
    ADD CONSTRAINT fk_billing_change_requests_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;

ALTER TABLE ONLY billing_checkout_sessions
    ADD CONSTRAINT fk_billing_checkout_sessions_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;

ALTER TABLE ONLY billing_subscriptions
    ADD CONSTRAINT fk_billing_subscriptions_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;

ALTER TABLE ONLY brands
    ADD CONSTRAINT fk_brands_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE RESTRICT;

ALTER TABLE ONLY competitor_observations
    ADD CONSTRAINT fk_competitor_observations_brand FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE;

ALTER TABLE ONLY competitors
    ADD CONSTRAINT fk_competitors_brand FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE;

ALTER TABLE ONLY generation_runs
    ADD CONSTRAINT fk_generation_runs_brand FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE;

ALTER TABLE ONLY memberships
    ADD CONSTRAINT fk_memberships_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE ONLY memberships
    ADD CONSTRAINT fk_memberships_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY password_reset_tokens
    ADD CONSTRAINT fk_password_reset_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY prompts
    ADD CONSTRAINT fk_prompts_brand FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE;

ALTER TABLE ONLY visibility_results
    ADD CONSTRAINT fk_visibility_results_run FOREIGN KEY (run_id) REFERENCES visibility_runs(id) ON DELETE CASCADE;

ALTER TABLE ONLY visibility_runs
    ADD CONSTRAINT fk_visibility_runs_brand FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE;

-- +goose Down
DROP TABLE IF EXISTS visibility_runs CASCADE;
DROP TABLE IF EXISTS visibility_results CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS stripe_webhook_events CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS prompts CASCADE;
DROP TABLE IF EXISTS password_reset_tokens CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS oauth_codes CASCADE;
DROP TABLE IF EXISTS oauth_clients CASCADE;
DROP TABLE IF EXISTS memberships CASCADE;
DROP TABLE IF EXISTS generation_runs CASCADE;
DROP TABLE IF EXISTS credentials CASCADE;
DROP TABLE IF EXISTS competitors CASCADE;
DROP TABLE IF EXISTS competitor_observations CASCADE;
DROP TABLE IF EXISTS brands CASCADE;
DROP TABLE IF EXISTS billing_subscriptions CASCADE;
DROP TABLE IF EXISTS billing_checkout_sessions CASCADE;
DROP TABLE IF EXISTS billing_change_requests CASCADE;
