-- Keygate Database Schema - Complete initialization
-- Consolidated schema for fresh installs, aligned with internal/model.

--
--

--
-- Name: feed_gating_lock(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.feed_gating_lock(pid text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtext('feed_gating:' || pid));
END $$;

--
-- Name: feed_gating_required(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.feed_gating_required(pid text) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
DECLARE gated BOOLEAN; ptype TEXT;
BEGIN
    SELECT feed_license_required, type INTO gated, ptype FROM products WHERE id = pid;
    RETURN ptype IN ('desktop', 'hybrid') AND NOT COALESCE(gated, false);
END $$;

--
-- Name: licenses_init_updates_until(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.licenses_init_updates_until() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE new_type TEXT; new_days INT; old_type TEXT;
BEGIN
    SELECT license_type, updates_days INTO new_type, new_days FROM plans WHERE id = NEW.plan_id;
    -- A plan change across the perpetual boundary decides the period
    -- again, whoever writes it. A replica that predates this column
    -- moves plan_id alone: leaving a perpetual plan would strand the
    -- old cutoff on a subscription (blocking the feed gate from ever
    -- coming off, and the migration from rolling back), and entering
    -- one again would keep that stale date instead of the new plan's
    -- period. The rule is the same one the application applies, so a
    -- current replica writing both columns agrees with it.
    IF TG_OP = 'UPDATE' AND OLD.plan_id IS DISTINCT FROM NEW.plan_id THEN
        SELECT license_type INTO old_type FROM plans WHERE id = OLD.plan_id;
        IF new_type IS DISTINCT FROM 'perpetual' THEN
            -- Leaving perpetual: the period belonged to that plan;
            -- subscriptions follow valid_until. The renewals bought
            -- into it are closed with it, as the application does: a
            -- refund of one of them must not later be subtracted from
            -- a period this licence earned somewhere else.
            NEW.updates_until := NULL;
            NEW.updates_terms_set := true;
            UPDATE license_renewals SET superseded_at = now()
             WHERE license_id = NEW.id AND superseded_at IS NULL;
            RETURN NEW;
        END IF;
        IF old_type IS DISTINCT FROM 'perpetual' THEN
            -- Entering perpetual: the new plan's period, whatever the
            -- row happened to carry before, and a ledger that belongs
            -- to no period this licence still has.
            IF COALESCE(new_days, 0) > 0 THEN
                NEW.updates_until := now() + make_interval(days => new_days);
            ELSE
                NEW.updates_until := NULL;
            END IF;
            NEW.updates_terms_set := true;
            UPDATE license_renewals SET superseded_at = now()
             WHERE license_id = NEW.id AND superseded_at IS NULL;
            RETURN NEW;
        END IF;
        -- Between two perpetual plans the customer keeps what they have.
        RETURN NEW;
    END IF;
    IF NEW.updates_until IS NOT NULL OR NEW.updates_terms_set THEN
        RETURN NEW;
    END IF;
    IF new_type IS DISTINCT FROM 'perpetual' OR COALESCE(new_days, 0) <= 0 THEN
        RETURN NEW;
    END IF;
    NEW.updates_until := now() + make_interval(days => new_days);
    RETURN NEW;
END $$;

--
-- Name: licenses_require_gated_feed(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.licenses_require_gated_feed() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.updates_until IS NOT NULL AND (TG_OP = 'INSERT' OR OLD.updates_until IS DISTINCT FROM NEW.updates_until OR OLD.product_id <> NEW.product_id) THEN
        PERFORM feed_gating_lock(NEW.product_id);
        IF feed_gating_required(NEW.product_id) THEN
            RAISE EXCEPTION 'licenses_feed_not_gated: license with an update cutoff on product % whose feeds are public', NEW.product_id
                USING ERRCODE = 'check_violation', CONSTRAINT = 'licenses_feed_not_gated';
        END IF;
    END IF;
    RETURN NEW;
END $$;

--
-- Name: plans_prices_disjoint(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.plans_prices_disjoint() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE price TEXT;
BEGIN
    FOR price IN
        SELECT p FROM unnest(ARRAY[NEW.stripe_price_id, NEW.stripe_renewal_price_id]) AS p
         WHERE p <> '' ORDER BY p
    LOOP
        PERFORM pg_advisory_xact_lock(hashtext('stripe_price:' || price));
    END LOOP;
    IF NEW.stripe_renewal_price_id <> '' AND (
        NEW.stripe_renewal_price_id = NEW.stripe_price_id
        OR EXISTS (SELECT 1 FROM plans WHERE stripe_price_id = NEW.stripe_renewal_price_id AND id <> NEW.id)
    ) THEN
        RAISE EXCEPTION 'plans_renewal_price_disjoint: stripe_renewal_price_id % is a purchase price of a plan', NEW.stripe_renewal_price_id
            USING ERRCODE = 'check_violation', CONSTRAINT = 'plans_renewal_price_disjoint';
    END IF;
    IF NEW.stripe_price_id <> '' AND EXISTS (
        SELECT 1 FROM plans WHERE stripe_renewal_price_id = NEW.stripe_price_id AND id <> NEW.id
    ) THEN
        RAISE EXCEPTION 'plans_renewal_price_disjoint: stripe_price_id % is a renewal price of a plan', NEW.stripe_price_id
            USING ERRCODE = 'check_violation', CONSTRAINT = 'plans_renewal_price_disjoint';
    END IF;
    RETURN NEW;
END $$;

--
-- Name: plans_record_update_terms(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.plans_record_update_terms() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO plan_update_terms (id, plan_id, updates_days, license_type, effective_from, recorded_at)
        VALUES (gen_random_uuid()::TEXT, NEW.id, NEW.updates_days, NEW.license_type,
                date_trunc('second', clock_timestamp()), clock_timestamp());
        RETURN NEW;
    END IF;
    IF NEW.updates_days IS DISTINCT FROM OLD.updates_days
       OR NEW.license_type IS DISTINCT FROM OLD.license_type THEN
        INSERT INTO plan_update_terms (id, plan_id, updates_days, license_type, effective_from, recorded_at)
        VALUES (gen_random_uuid()::TEXT, NEW.id, NEW.updates_days, NEW.license_type,
                date_trunc('second', clock_timestamp()) + interval '1 second', clock_timestamp());
    END IF;
    RETURN NEW;
END $$;

--
-- Name: plans_require_gated_feed(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.plans_require_gated_feed() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.updates_days > 0 OR NEW.stripe_renewal_price_id <> '' THEN
        PERFORM feed_gating_lock(NEW.product_id);
        IF feed_gating_required(NEW.product_id) THEN
            RAISE EXCEPTION 'plans_feed_not_gated: plan with an update period or renewals on product % whose feeds are public', NEW.product_id
                USING ERRCODE = 'check_violation', CONSTRAINT = 'plans_feed_not_gated';
        END IF;
    END IF;
    RETURN NEW;
END $$;

--
-- Name: products_keep_feed_gated(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.products_keep_feed_gated() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- The instant the gate went up is what the wait before an update
    -- period may be sold is measured from, and a NULL there reads as
    -- "nothing to wait out". The application stamps it; a direct
    -- UPDATE by an operator or an older tool would not, so it is
    -- stamped here too — from the database clock, in the same
    -- statement that switches the gate on.
    IF TG_OP = 'INSERT' THEN
        IF NEW.feed_license_required AND NEW.feed_gated_at IS NULL THEN
            NEW.feed_gated_at := clock_timestamp();
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.feed_license_required AND NOT OLD.feed_license_required
       AND NEW.feed_gated_at IS NOT DISTINCT FROM OLD.feed_gated_at THEN
        NEW.feed_gated_at := clock_timestamp();
    END IF;
    IF NEW.type IN ('desktop', 'hybrid') AND NOT NEW.feed_license_required
       AND (OLD.feed_license_required OR OLD.type NOT IN ('desktop', 'hybrid')) THEN
        PERFORM feed_gating_lock(NEW.id);
        IF EXISTS (SELECT 1 FROM plans WHERE product_id = NEW.id AND (updates_days > 0 OR stripe_renewal_price_id <> ''))
           OR EXISTS (SELECT 1 FROM licenses WHERE product_id = NEW.id AND updates_until IS NOT NULL) THEN
            RAISE EXCEPTION 'products_feed_gate_in_use: product % has plans or licenses with an update period', NEW.id
                USING ERRCODE = 'check_violation', CONSTRAINT = 'products_feed_gate_in_use';
        END IF;
    END IF;
    RETURN NEW;
END $$;

--
-- Name: activations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.activations (
    id text NOT NULL,
    license_id text NOT NULL,
    identifier text NOT NULL,
    identifier_type text NOT NULL,
    label text DEFAULT ''::text NOT NULL,
    ip_address text DEFAULT ''::text NOT NULL,
    last_verified timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT activations_identifier_type_check CHECK ((identifier_type = ANY (ARRAY['device'::text, 'user'::text])))
);

--
-- Name: addons; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.addons (
    id text NOT NULL,
    product_id text NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    feature text NOT NULL,
    value_type text NOT NULL,
    value text NOT NULL,
    quota_period text DEFAULT ''::text NOT NULL,
    quota_unit text DEFAULT ''::text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT addons_value_type_check CHECK ((value_type = ANY (ARRAY['bool'::text, 'int'::text, 'string'::text, 'quota'::text])))
);

--
-- Name: analytics_snapshots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.analytics_snapshots (
    id text NOT NULL,
    date date NOT NULL,
    product_id text NOT NULL,
    total_licenses integer DEFAULT 0 NOT NULL,
    active_licenses integer DEFAULT 0 NOT NULL,
    new_licenses integer DEFAULT 0 NOT NULL,
    churned integer DEFAULT 0 NOT NULL,
    total_activations integer DEFAULT 0 NOT NULL,
    total_seats integer DEFAULT 0 NOT NULL,
    total_usage bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: api_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_keys (
    id text NOT NULL,
    product_id text,
    name text NOT NULL,
    key_hash text NOT NULL,
    prefix text NOT NULL,
    scopes text[] DEFAULT '{}'::text[] NOT NULL,
    last_used timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_ip text DEFAULT ''::text NOT NULL
);

--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_logs (
    id text NOT NULL,
    entity text NOT NULL,
    entity_id text NOT NULL,
    action text NOT NULL,
    actor_id text DEFAULT ''::text NOT NULL,
    actor_type text DEFAULT ''::text NOT NULL,
    changes jsonb DEFAULT '{}'::jsonb NOT NULL,
    ip_address text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: email_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_queue (
    id text NOT NULL,
    to_addr text NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    max_attempts integer DEFAULT 5 NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    next_retry timestamp with time zone,
    error text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sent_at timestamp with time zone,
    notification_id text DEFAULT ''::text NOT NULL,
    claim_token text DEFAULT ''::text NOT NULL,
    CONSTRAINT email_queue_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'sent'::text, 'failed'::text])))
);

--
-- Name: entitlements; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entitlements (
    id text NOT NULL,
    plan_id text NOT NULL,
    feature text NOT NULL,
    value_type text NOT NULL,
    value text DEFAULT ''::text NOT NULL,
    quota_period text DEFAULT ''::text NOT NULL,
    quota_unit text DEFAULT ''::text NOT NULL,
    stripe_meter_event_name text DEFAULT ''::text NOT NULL,
    CONSTRAINT entitlements_quota_period_check CHECK ((quota_period = ANY (ARRAY[''::text, 'hourly'::text, 'daily'::text, 'monthly'::text, 'yearly'::text]))),
    CONSTRAINT entitlements_value_type_check CHECK ((value_type = ANY (ARRAY['bool'::text, 'int'::text, 'string'::text, 'quota'::text, 'flag'::text])))
);

--
-- Name: floating_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.floating_sessions (
    id text NOT NULL,
    license_id text NOT NULL,
    identifier text NOT NULL,
    label text DEFAULT ''::text NOT NULL,
    ip_address text DEFAULT ''::text NOT NULL,
    checked_out timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    heartbeat timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: idempotency_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.idempotency_keys (
    key text NOT NULL,
    endpoint text NOT NULL,
    body_hash text NOT NULL,
    response_status integer DEFAULT 0 NOT NULL,
    response_body text DEFAULT ''::text NOT NULL,
    response_complete boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone DEFAULT (now() + '24:00:00'::interval) NOT NULL,
    CONSTRAINT idempotency_keys_body_hash_check CHECK ((body_hash ~ '^[a-f0-9]{64}$'::text)),
    CONSTRAINT idempotency_keys_endpoint_check CHECK (((length(endpoint) >= 1) AND (length(endpoint) <= 256))),
    CONSTRAINT idempotency_keys_key_check CHECK (((length(key) >= 1) AND (length(key) <= 256)))
);

--
-- Name: license_addons; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.license_addons (
    id text NOT NULL,
    license_id text NOT NULL,
    addon_id text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: license_renewals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.license_renewals (
    id text NOT NULL,
    license_id text NOT NULL,
    stripe_checkout_session_id text DEFAULT ''::text NOT NULL,
    stripe_payment_intent_id text DEFAULT ''::text NOT NULL,
    days integer NOT NULL,
    previous_updates_until timestamp with time zone,
    updates_until timestamp with time zone,
    refunded_at timestamp with time zone,
    superseded_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: licenses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.licenses (
    id text NOT NULL,
    product_id text NOT NULL,
    plan_id text NOT NULL,
    user_id text,
    email text NOT NULL,
    license_key text NOT NULL,
    payment_provider text DEFAULT ''::text NOT NULL,
    stripe_customer_id text DEFAULT ''::text NOT NULL,
    stripe_subscription_id text,
    status text DEFAULT 'active'::text NOT NULL,
    valid_from timestamp with time zone DEFAULT now() NOT NULL,
    valid_until timestamp with time zone,
    canceled_at timestamp with time zone,
    suspended_at timestamp with time zone,
    notes text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    org_name text DEFAULT ''::text NOT NULL,
    key_hash text DEFAULT ''::text NOT NULL,
    license_key_encrypted bytea,
    external_customer_id text DEFAULT ''::text NOT NULL,
    external_workspace_id text DEFAULT ''::text NOT NULL,
    past_due_at timestamp with time zone,
    stripe_payment_intent_id text DEFAULT ''::text NOT NULL,
    stripe_checkout_session_id text DEFAULT ''::text NOT NULL,
    updates_until timestamp with time zone,
    updates_terms_set boolean DEFAULT false NOT NULL,
    stripe_synced_at timestamp with time zone,
    suspended_by text,
    CONSTRAINT licenses_license_key_encrypted_check CHECK (((license_key_encrypted IS NULL) OR ((octet_length(license_key_encrypted) >= 28) AND (octet_length(license_key_encrypted) <= 4096)))),
    CONSTRAINT licenses_status_check CHECK ((status = ANY (ARRAY['active'::text, 'trialing'::text, 'past_due'::text, 'canceled'::text, 'expired'::text, 'suspended'::text, 'revoked'::text])))
);

--
-- Name: metered_billing; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.metered_billing (
    id text NOT NULL,
    license_id text NOT NULL,
    feature text NOT NULL,
    quantity bigint NOT NULL,
    period_key text NOT NULL,
    synced boolean DEFAULT false NOT NULL,
    synced_at timestamp with time zone,
    external_id text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    identifier text DEFAULT ''::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_error text DEFAULT ''::text NOT NULL
);

--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id text NOT NULL,
    license_id text NOT NULL,
    tag text NOT NULL,
    sent_at timestamp with time zone DEFAULT now(),
    claimed_at timestamp with time zone
);

--
-- Name: oauth_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_accounts (
    id text NOT NULL,
    user_id text NOT NULL,
    provider text NOT NULL,
    provider_id text NOT NULL,
    email text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: otp_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.otp_codes (
    id text NOT NULL,
    email text NOT NULL,
    code_hash text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: payment_transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.payment_transactions (
    id bigint NOT NULL,
    provider_name text NOT NULL,
    provider_tx_id text NOT NULL,
    session_id text DEFAULT ''::text NOT NULL,
    license_id text,
    amount bigint NOT NULL,
    currency text DEFAULT 'USD'::text NOT NULL,
    status text NOT NULL,
    payment_method text DEFAULT ''::text NOT NULL,
    customer_email text DEFAULT ''::text NOT NULL,
    refund_amount bigint DEFAULT 0 NOT NULL,
    refunded_at timestamp with time zone,
    metadata text DEFAULT ''::text NOT NULL,
    processed_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT payment_transactions_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'completed'::text, 'failed'::text, 'cancelled'::text, 'refunded'::text])))
);

--
-- Name: payment_transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.payment_transactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: payment_transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.payment_transactions_id_seq OWNED BY public.payment_transactions.id;

--
-- Name: plan_update_terms; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plan_update_terms (
    id text NOT NULL,
    plan_id text NOT NULL,
    updates_days integer NOT NULL,
    license_type text DEFAULT ''::text NOT NULL,
    effective_from timestamp with time zone DEFAULT (date_trunc('second'::text, now()) + '00:00:01'::interval) NOT NULL,
    recorded_at timestamp with time zone DEFAULT clock_timestamp() NOT NULL
);

--
-- Name: plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plans (
    id text NOT NULL,
    product_id text NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    license_type text NOT NULL,
    billing_interval text DEFAULT ''::text NOT NULL,
    max_activations integer DEFAULT 3 NOT NULL,
    trial_days integer DEFAULT 0 NOT NULL,
    grace_days integer DEFAULT 7 NOT NULL,
    stripe_price_id text DEFAULT ''::text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    max_seats integer DEFAULT 0 NOT NULL,
    license_model text DEFAULT 'standard'::text NOT NULL,
    floating_timeout integer DEFAULT 30 NOT NULL,
    checkout_id text NOT NULL,
    token_ttl_days integer DEFAULT 0 NOT NULL,
    updates_days integer DEFAULT 0 NOT NULL,
    renewal_days integer DEFAULT 0 NOT NULL,
    stripe_renewal_price_id text DEFAULT ''::text NOT NULL,
    price_cny integer,
    price_hkd integer,
    price_usd integer,
    CONSTRAINT plans_billing_interval_check CHECK ((billing_interval = ANY (ARRAY[''::text, 'month'::text, 'year'::text]))),
    CONSTRAINT plans_license_model_check CHECK ((license_model = ANY (ARRAY['standard'::text, 'floating'::text]))),
    CONSTRAINT plans_license_type_check CHECK ((license_type = ANY (ARRAY['subscription'::text, 'perpetual'::text, 'trial'::text])))
);

--
-- Name: processed_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.processed_events (
    id text NOT NULL,
    provider text NOT NULL,
    event_id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: products; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.products (
    id text NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    type text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    minimum_supported_version text DEFAULT ''::text NOT NULL,
    minimum_supported_message text DEFAULT ''::text NOT NULL,
    require_signing boolean DEFAULT true NOT NULL,
    feed_license_required boolean DEFAULT false NOT NULL,
    feed_gated_at timestamp with time zone,
    download_url text DEFAULT ''::text NOT NULL,
    CONSTRAINT products_minimum_supported_message_check CHECK ((length(minimum_supported_message) <= 1024)),
    CONSTRAINT products_minimum_supported_version_check CHECK (((minimum_supported_version = ''::text) OR (minimum_supported_version ~ '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'::text))),
    CONSTRAINT products_type_check CHECK ((type = ANY (ARRAY['desktop'::text, 'saas'::text, 'hybrid'::text])))
);

--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.refresh_tokens (
    id text NOT NULL,
    user_id text NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    revoked_at timestamp with time zone
);

--
-- Name: release_artifacts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.release_artifacts (
    id text NOT NULL,
    release_id text NOT NULL,
    platform text NOT NULL,
    file_key text DEFAULT ''::text NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    sha256 text DEFAULT ''::text NOT NULL,
    ed25519_sig text DEFAULT ''::text NOT NULL,
    content_type text DEFAULT 'application/octet-stream'::text NOT NULL,
    signing_key_id text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    filename text DEFAULT ''::text NOT NULL,
    tauri_signature text DEFAULT ''::text NOT NULL,
    CONSTRAINT release_artifacts_content_type_check CHECK ((length(content_type) <= 128)),
    CONSTRAINT release_artifacts_ed25519_sig_check CHECK ((length(ed25519_sig) <= 256)),
    CONSTRAINT release_artifacts_file_key_check CHECK ((length(file_key) <= 1024)),
    CONSTRAINT release_artifacts_file_size_check CHECK ((file_size >= 0)),
    CONSTRAINT release_artifacts_platform_check CHECK (((length(platform) >= 1) AND (length(platform) <= 64))),
    CONSTRAINT release_artifacts_sha256_check CHECK (((sha256 = ''::text) OR (sha256 ~ '^[a-f0-9]{64}$'::text)))
);

--
-- Name: release_signing_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.release_signing_keys (
    id text NOT NULL,
    product_id text NOT NULL,
    public_key text NOT NULL,
    private_key_encrypted bytea NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    rotated_at timestamp with time zone,
    note text DEFAULT ''::text NOT NULL,
    CONSTRAINT release_signing_keys_note_check CHECK ((length(note) <= 256)),
    CONSTRAINT release_signing_keys_private_key_encrypted_check CHECK (((octet_length(private_key_encrypted) >= 60) AND (octet_length(private_key_encrypted) <= 256))),
    CONSTRAINT release_signing_keys_public_key_check CHECK (((length(public_key) >= 32) AND (length(public_key) <= 128)))
);

--
-- Name: releases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.releases (
    id text CONSTRAINT releases_id_not_null1 NOT NULL,
    product_id text CONSTRAINT releases_product_id_not_null1 NOT NULL,
    version text CONSTRAINT releases_version_not_null1 NOT NULL,
    channel text DEFAULT 'stable'::text CONSTRAINT releases_channel_not_null1 NOT NULL,
    name text DEFAULT ''::text CONSTRAINT releases_name_not_null1 NOT NULL,
    release_notes text DEFAULT ''::text CONSTRAINT releases_release_notes_not_null1 NOT NULL,
    status text DEFAULT 'draft'::text CONSTRAINT releases_status_not_null1 NOT NULL,
    yanked_reason text DEFAULT ''::text CONSTRAINT releases_yanked_reason_not_null1 NOT NULL,
    published_at timestamp with time zone,
    yanked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() CONSTRAINT releases_created_at_not_null1 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() CONSTRAINT releases_updated_at_not_null1 NOT NULL,
    CONSTRAINT releases_channel_check1 CHECK ((channel = ANY (ARRAY['stable'::text, 'beta'::text, 'alpha'::text, 'dev'::text]))),
    CONSTRAINT releases_name_check1 CHECK ((length(name) <= 256)),
    CONSTRAINT releases_release_notes_check1 CHECK ((length(release_notes) <= 65536)),
    CONSTRAINT releases_status_check1 CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'yanked'::text]))),
    CONSTRAINT releases_version_check1 CHECK (((length(version) >= 1) AND (length(version) <= 64) AND (version ~ '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'::text))),
    CONSTRAINT releases_yanked_reason_check1 CHECK ((length(yanked_reason) <= 1024))
);

--
-- Name: releases_legacy; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.releases_legacy (
    id text CONSTRAINT releases_id_not_null NOT NULL,
    product_id text CONSTRAINT releases_product_id_not_null NOT NULL,
    version text CONSTRAINT releases_version_not_null NOT NULL,
    channel text DEFAULT 'stable'::text CONSTRAINT releases_channel_not_null NOT NULL,
    platform text CONSTRAINT releases_platform_not_null NOT NULL,
    file_key text DEFAULT ''::text CONSTRAINT releases_file_key_not_null NOT NULL,
    file_size bigint DEFAULT 0 CONSTRAINT releases_file_size_not_null NOT NULL,
    sha256 text DEFAULT ''::text CONSTRAINT releases_sha256_not_null NOT NULL,
    ed25519_sig text DEFAULT ''::text CONSTRAINT releases_ed25519_sig_not_null NOT NULL,
    content_type text DEFAULT 'application/octet-stream'::text CONSTRAINT releases_content_type_not_null NOT NULL,
    name text DEFAULT ''::text CONSTRAINT releases_name_not_null NOT NULL,
    release_notes text DEFAULT ''::text CONSTRAINT releases_release_notes_not_null NOT NULL,
    status text DEFAULT 'draft'::text CONSTRAINT releases_status_not_null NOT NULL,
    yanked_reason text DEFAULT ''::text CONSTRAINT releases_yanked_reason_not_null NOT NULL,
    published_at timestamp with time zone,
    yanked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() CONSTRAINT releases_created_at_not_null NOT NULL,
    updated_at timestamp with time zone DEFAULT now() CONSTRAINT releases_updated_at_not_null NOT NULL,
    signing_key_id text,
    CONSTRAINT releases_channel_check CHECK ((channel = ANY (ARRAY['stable'::text, 'beta'::text, 'alpha'::text, 'dev'::text]))),
    CONSTRAINT releases_content_type_check CHECK ((length(content_type) <= 128)),
    CONSTRAINT releases_ed25519_sig_check CHECK ((length(ed25519_sig) <= 256)),
    CONSTRAINT releases_file_key_check CHECK ((length(file_key) <= 1024)),
    CONSTRAINT releases_file_size_check CHECK ((file_size >= 0)),
    CONSTRAINT releases_name_check CHECK ((length(name) <= 256)),
    CONSTRAINT releases_platform_check CHECK (((length(platform) >= 1) AND (length(platform) <= 64))),
    CONSTRAINT releases_release_notes_check CHECK ((length(release_notes) <= 65536)),
    CONSTRAINT releases_sha256_check CHECK (((sha256 = ''::text) OR (sha256 ~ '^[a-f0-9]{64}$'::text))),
    CONSTRAINT releases_status_check CHECK ((status = ANY (ARRAY['draft'::text, 'published'::text, 'yanked'::text]))),
    CONSTRAINT releases_version_check CHECK (((length(version) >= 1) AND (length(version) <= 64) AND (version ~ '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'::text))),
    CONSTRAINT releases_yanked_reason_check CHECK ((length(yanked_reason) <= 1024))
);

--
-- Name: renewal_reminders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.renewal_reminders (
    id bigint NOT NULL,
    license_id text NOT NULL,
    days_before integer NOT NULL,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: renewal_reminders_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.renewal_reminders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: renewal_reminders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.renewal_reminders_id_seq OWNED BY public.renewal_reminders.id;

--
-- Name: seats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seats (
    id text NOT NULL,
    license_id text NOT NULL,
    user_id text,
    email text NOT NULL,
    role text DEFAULT 'member'::text NOT NULL,
    invited_at timestamp with time zone DEFAULT now() NOT NULL,
    accepted_at timestamp with time zone,
    removed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    invite_token_hash text,
    invite_expires_at timestamp with time zone,
    CONSTRAINT seats_role_check CHECK ((role = ANY (ARRAY['admin'::text, 'member'::text])))
);

--
-- Name: settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.settings (
    key text NOT NULL,
    value text DEFAULT ''::text NOT NULL
);

--
-- Name: subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.subscriptions (
    id text NOT NULL,
    license_id text NOT NULL,
    user_id text,
    plan_id text NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    payment_provider text DEFAULT ''::text NOT NULL,
    external_id text DEFAULT ''::text NOT NULL,
    current_period_start timestamp with time zone,
    current_period_end timestamp with time zone,
    cancel_at_period_end boolean DEFAULT false NOT NULL,
    canceled_at timestamp with time zone,
    trial_start timestamp with time zone,
    trial_end timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    cancel_state_synced_at timestamp with time zone,
    CONSTRAINT subscriptions_status_check CHECK ((status = ANY (ARRAY['active'::text, 'trialing'::text, 'past_due'::text, 'canceled'::text, 'expired'::text, 'paused'::text, 'suspended'::text, 'revoked'::text])))
);

--
-- Name: usage_counters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usage_counters (
    id text NOT NULL,
    license_id text NOT NULL,
    feature text NOT NULL,
    period text NOT NULL,
    period_key text NOT NULL,
    used bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: usage_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usage_events (
    id text NOT NULL,
    license_id text NOT NULL,
    feature text NOT NULL,
    quantity bigint DEFAULT 1 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    ip_address text DEFAULT ''::text NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id text NOT NULL,
    email text NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    avatar_url text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    role text DEFAULT 'user'::text NOT NULL,
    CONSTRAINT users_role_check CHECK ((role = ANY (ARRAY['owner'::text, 'admin'::text, 'user'::text])))
);

--
-- Name: webhook_deliveries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhook_deliveries (
    id text NOT NULL,
    webhook_id text NOT NULL,
    event text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    response_code integer,
    response_body text DEFAULT ''::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    next_retry timestamp with time zone,
    status text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    delivered_at timestamp with time zone,
    CONSTRAINT webhook_deliveries_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'delivered'::text, 'failed'::text])))
);

--
-- Name: webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhooks (
    id text NOT NULL,
    product_id text NOT NULL,
    url text NOT NULL,
    secret text NOT NULL,
    events text[] DEFAULT '{}'::text[] NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

--
-- Name: payment_transactions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_transactions ALTER COLUMN id SET DEFAULT nextval('public.payment_transactions_id_seq'::regclass);

--
-- Name: renewal_reminders id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.renewal_reminders ALTER COLUMN id SET DEFAULT nextval('public.renewal_reminders_id_seq'::regclass);

--
-- Name: activations activations_license_id_identifier_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activations
    ADD CONSTRAINT activations_license_id_identifier_key UNIQUE (license_id, identifier);

--
-- Name: activations activations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activations
    ADD CONSTRAINT activations_pkey PRIMARY KEY (id);

--
-- Name: addons addons_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.addons
    ADD CONSTRAINT addons_pkey PRIMARY KEY (id);

--
-- Name: addons addons_product_id_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.addons
    ADD CONSTRAINT addons_product_id_slug_key UNIQUE (product_id, slug);

--
-- Name: analytics_snapshots analytics_snapshots_date_product_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytics_snapshots
    ADD CONSTRAINT analytics_snapshots_date_product_id_key UNIQUE (date, product_id);

--
-- Name: analytics_snapshots analytics_snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytics_snapshots
    ADD CONSTRAINT analytics_snapshots_pkey PRIMARY KEY (id);

--
-- Name: api_keys api_keys_key_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash);

--
-- Name: api_keys api_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);

--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);

--
-- Name: email_queue email_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_queue
    ADD CONSTRAINT email_queue_pkey PRIMARY KEY (id);

--
-- Name: entitlements entitlements_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entitlements
    ADD CONSTRAINT entitlements_pkey PRIMARY KEY (id);

--
-- Name: entitlements entitlements_plan_id_feature_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entitlements
    ADD CONSTRAINT entitlements_plan_id_feature_key UNIQUE (plan_id, feature);

--
-- Name: floating_sessions floating_sessions_license_id_identifier_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.floating_sessions
    ADD CONSTRAINT floating_sessions_license_id_identifier_key UNIQUE (license_id, identifier);

--
-- Name: floating_sessions floating_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.floating_sessions
    ADD CONSTRAINT floating_sessions_pkey PRIMARY KEY (id);

--
-- Name: idempotency_keys idempotency_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.idempotency_keys
    ADD CONSTRAINT idempotency_keys_pkey PRIMARY KEY (key, endpoint);

--
-- Name: license_addons license_addons_license_id_addon_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.license_addons
    ADD CONSTRAINT license_addons_license_id_addon_id_key UNIQUE (license_id, addon_id);

--
-- Name: license_addons license_addons_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.license_addons
    ADD CONSTRAINT license_addons_pkey PRIMARY KEY (id);

--
-- Name: license_renewals license_renewals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.license_renewals
    ADD CONSTRAINT license_renewals_pkey PRIMARY KEY (id);

--
-- Name: licenses licenses_license_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_license_key_key UNIQUE (license_key);

--
-- Name: licenses licenses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_pkey PRIMARY KEY (id);

--
-- Name: licenses licenses_stripe_subscription_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_stripe_subscription_id_key UNIQUE (stripe_subscription_id);

--
-- Name: metered_billing metered_billing_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.metered_billing
    ADD CONSTRAINT metered_billing_pkey PRIMARY KEY (id);

--
-- Name: notifications notifications_license_id_tag_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_license_id_tag_key UNIQUE (license_id, tag);

--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);

--
-- Name: oauth_accounts oauth_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_accounts
    ADD CONSTRAINT oauth_accounts_pkey PRIMARY KEY (id);

--
-- Name: oauth_accounts oauth_accounts_provider_provider_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_accounts
    ADD CONSTRAINT oauth_accounts_provider_provider_id_key UNIQUE (provider, provider_id);

--
-- Name: otp_codes otp_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.otp_codes
    ADD CONSTRAINT otp_codes_pkey PRIMARY KEY (id);

--
-- Name: payment_transactions payment_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_pkey PRIMARY KEY (id);

--
-- Name: plan_update_terms plan_update_terms_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_update_terms
    ADD CONSTRAINT plan_update_terms_pkey PRIMARY KEY (id);

--
-- Name: plans plans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT plans_pkey PRIMARY KEY (id);

--
-- Name: plans plans_product_id_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT plans_product_id_slug_key UNIQUE (product_id, slug);

--
-- Name: processed_events processed_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.processed_events
    ADD CONSTRAINT processed_events_pkey PRIMARY KEY (id);

--
-- Name: processed_events processed_events_provider_event_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.processed_events
    ADD CONSTRAINT processed_events_provider_event_id_key UNIQUE (provider, event_id);

--
-- Name: products products_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_pkey PRIMARY KEY (id);

--
-- Name: products products_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_slug_key UNIQUE (slug);

--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);

--
-- Name: refresh_tokens refresh_tokens_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash);

--
-- Name: release_artifacts release_artifacts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release_artifacts
    ADD CONSTRAINT release_artifacts_pkey PRIMARY KEY (id);

--
-- Name: release_artifacts release_artifacts_release_id_platform_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release_artifacts
    ADD CONSTRAINT release_artifacts_release_id_platform_key UNIQUE (release_id, platform);

--
-- Name: release_signing_keys release_signing_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release_signing_keys
    ADD CONSTRAINT release_signing_keys_pkey PRIMARY KEY (id);

--
-- Name: releases_legacy releases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.releases_legacy
    ADD CONSTRAINT releases_pkey PRIMARY KEY (id);

--
-- Name: releases releases_pkey1; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.releases
    ADD CONSTRAINT releases_pkey1 PRIMARY KEY (id);

--
-- Name: releases releases_product_id_version_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.releases
    ADD CONSTRAINT releases_product_id_version_key UNIQUE (product_id, version);

--
-- Name: releases_legacy releases_product_id_version_platform_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.releases_legacy
    ADD CONSTRAINT releases_product_id_version_platform_key UNIQUE (product_id, version, platform);

--
-- Name: renewal_reminders renewal_reminders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.renewal_reminders
    ADD CONSTRAINT renewal_reminders_pkey PRIMARY KEY (id);

--
-- Name: seats seats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seats
    ADD CONSTRAINT seats_pkey PRIMARY KEY (id);

--
-- Name: settings settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (key);

--
-- Name: subscriptions subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_pkey PRIMARY KEY (id);

--
-- Name: renewal_reminders unique_license_reminder_days; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.renewal_reminders
    ADD CONSTRAINT unique_license_reminder_days UNIQUE (license_id, days_before);

--
-- Name: payment_transactions unique_provider_transaction; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT unique_provider_transaction UNIQUE (provider_name, provider_tx_id);

--
-- Name: usage_counters usage_counters_license_id_feature_period_period_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_counters
    ADD CONSTRAINT usage_counters_license_id_feature_period_period_key_key UNIQUE (license_id, feature, period, period_key);

--
-- Name: usage_counters usage_counters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_counters
    ADD CONSTRAINT usage_counters_pkey PRIMARY KEY (id);

--
-- Name: usage_events usage_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_events
    ADD CONSTRAINT usage_events_pkey PRIMARY KEY (id);

--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);

--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

--
-- Name: webhook_deliveries webhook_deliveries_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_deliveries
    ADD CONSTRAINT webhook_deliveries_pkey PRIMARY KEY (id);

--
-- Name: webhooks webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_pkey PRIMARY KEY (id);

--
-- Name: idx_activations_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_activations_license ON public.activations USING btree (license_id);

--
-- Name: idx_addons_product; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_addons_product ON public.addons USING btree (product_id);

--
-- Name: idx_analytics_product_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_analytics_product_date ON public.analytics_snapshots USING btree (product_id, date);

--
-- Name: idx_apikeys_product; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_apikeys_product ON public.api_keys USING btree (product_id);

--
-- Name: idx_audit_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_created ON public.audit_logs USING btree (created_at);

--
-- Name: idx_audit_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_entity ON public.audit_logs USING btree (entity, entity_id);

--
-- Name: idx_counters_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_counters_lookup ON public.usage_counters USING btree (license_id, feature, period, period_key);

--
-- Name: idx_deliveries_retry; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deliveries_retry ON public.webhook_deliveries USING btree (status, next_retry) WHERE (status = 'pending'::text);

--
-- Name: idx_deliveries_webhook; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_deliveries_webhook ON public.webhook_deliveries USING btree (webhook_id);

--
-- Name: idx_email_queue_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_queue_status ON public.email_queue USING btree (status, next_retry) WHERE (status = 'pending'::text);

--
-- Name: idx_floating_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_floating_expires ON public.floating_sessions USING btree (expires_at);

--
-- Name: idx_floating_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_floating_license ON public.floating_sessions USING btree (license_id);

--
-- Name: idx_idempotency_keys_expires_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_idempotency_keys_expires_at ON public.idempotency_keys USING btree (expires_at);

--
-- Name: idx_license_addons_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_license_addons_license ON public.license_addons USING btree (license_id);

--
-- Name: idx_license_renewals_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_license_renewals_license ON public.license_renewals USING btree (license_id);

--
-- Name: idx_license_renewals_payment_intent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_license_renewals_payment_intent ON public.license_renewals USING btree (stripe_payment_intent_id) WHERE (stripe_payment_intent_id <> ''::text);

--
-- Name: idx_license_renewals_session; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_license_renewals_session ON public.license_renewals USING btree (stripe_checkout_session_id) WHERE (stripe_checkout_session_id <> ''::text);

--
-- Name: idx_licenses_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_email ON public.licenses USING btree (email);

--
-- Name: idx_licenses_email_lower; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_email_lower ON public.licenses USING btree (lower(email));

--
-- Name: idx_licenses_external_customer; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_external_customer ON public.licenses USING btree (product_id, external_customer_id) WHERE (external_customer_id <> ''::text);

--
-- Name: idx_licenses_external_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_external_workspace ON public.licenses USING btree (product_id, external_workspace_id) WHERE (external_workspace_id <> ''::text);

--
-- Name: idx_licenses_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_key ON public.licenses USING btree (license_key);

--
-- Name: idx_licenses_key_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_licenses_key_hash ON public.licenses USING btree (key_hash) WHERE (key_hash <> ''::text);

--
-- Name: idx_licenses_past_due_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_past_due_at ON public.licenses USING btree (past_due_at) WHERE (status = 'past_due'::text);

--
-- Name: idx_licenses_product_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_product_status ON public.licenses USING btree (product_id, status);

--
-- Name: idx_licenses_stripe_checkout_session; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_licenses_stripe_checkout_session ON public.licenses USING btree (stripe_checkout_session_id) WHERE (stripe_checkout_session_id <> ''::text);

--
-- Name: idx_licenses_stripe_payment_intent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_stripe_payment_intent ON public.licenses USING btree (stripe_payment_intent_id) WHERE (stripe_payment_intent_id <> ''::text);

--
-- Name: idx_licenses_stripe_sub; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_stripe_sub ON public.licenses USING btree (stripe_subscription_id);

--
-- Name: idx_licenses_updates_until; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_updates_until ON public.licenses USING btree (updates_until) WHERE (updates_until IS NOT NULL);

--
-- Name: idx_licenses_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_user ON public.licenses USING btree (user_id);

--
-- Name: idx_licenses_valid_until; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_licenses_valid_until ON public.licenses USING btree (valid_until) WHERE (valid_until IS NOT NULL);

--
-- Name: idx_metered_billing_identifier; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_metered_billing_identifier ON public.metered_billing USING btree (identifier) WHERE (identifier <> ''::text);

--
-- Name: idx_metered_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_metered_license ON public.metered_billing USING btree (license_id);

--
-- Name: idx_metered_unsynced; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_metered_unsynced ON public.metered_billing USING btree (synced) WHERE (synced = false);

--
-- Name: idx_notifications_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_license ON public.notifications USING btree (license_id);

--
-- Name: idx_oauth_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_oauth_user ON public.oauth_accounts USING btree (user_id);

--
-- Name: idx_otp_codes_email_expires; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_otp_codes_email_expires ON public.otp_codes USING btree (email, expires_at);

--
-- Name: idx_payment_transactions_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_transactions_created_at ON public.payment_transactions USING btree (created_at DESC);

--
-- Name: idx_payment_transactions_customer_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_transactions_customer_email ON public.payment_transactions USING btree (customer_email);

--
-- Name: idx_payment_transactions_license_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_transactions_license_id ON public.payment_transactions USING btree (license_id);

--
-- Name: idx_payment_transactions_provider_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_payment_transactions_provider_status ON public.payment_transactions USING btree (provider_name, status);

--
-- Name: idx_plan_update_terms_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plan_update_terms_lookup ON public.plan_update_terms USING btree (plan_id, effective_from DESC, recorded_at DESC);

--
-- Name: idx_plans_checkout_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_plans_checkout_id ON public.plans USING btree (checkout_id);

--
-- Name: idx_plans_stripe_price_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_plans_stripe_price_unique ON public.plans USING btree (stripe_price_id) WHERE (stripe_price_id <> ''::text);

--
-- Name: idx_processed_events_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_processed_events_lookup ON public.processed_events USING btree (provider, event_id);

--
-- Name: idx_refresh_tokens_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_hash ON public.refresh_tokens USING btree (token_hash);

--
-- Name: idx_refresh_tokens_revoked; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_revoked ON public.refresh_tokens USING btree (revoked_at) WHERE (revoked_at IS NOT NULL);

--
-- Name: idx_refresh_tokens_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_refresh_tokens_user ON public.refresh_tokens USING btree (user_id);

--
-- Name: idx_release_artifacts_platform; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_release_artifacts_platform ON public.release_artifacts USING btree (platform);

--
-- Name: idx_release_artifacts_release; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_release_artifacts_release ON public.release_artifacts USING btree (release_id);

--
-- Name: idx_release_signing_keys_one_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_release_signing_keys_one_active ON public.release_signing_keys USING btree (product_id) WHERE (active = true);

--
-- Name: idx_release_signing_keys_product; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_release_signing_keys_product ON public.release_signing_keys USING btree (product_id, active, created_at DESC);

--
-- Name: idx_releases_drafts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_releases_drafts ON public.releases USING btree (product_id, created_at DESC) WHERE (status = 'draft'::text);

--
-- Name: idx_releases_legacy_drafts; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_releases_legacy_drafts ON public.releases_legacy USING btree (product_id, created_at DESC) WHERE (status = 'draft'::text);

--
-- Name: idx_releases_legacy_published; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_releases_legacy_published ON public.releases_legacy USING btree (product_id, channel, platform, published_at DESC) WHERE (status = 'published'::text);

--
-- Name: idx_releases_legacy_signing_key; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_releases_legacy_signing_key ON public.releases_legacy USING btree (signing_key_id) WHERE (signing_key_id IS NOT NULL);

--
-- Name: idx_releases_published; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_releases_published ON public.releases USING btree (product_id, channel, published_at DESC) WHERE (status = 'published'::text);

--
-- Name: idx_seats_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seats_email ON public.seats USING btree (email);

--
-- Name: idx_seats_email_lower; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seats_email_lower ON public.seats USING btree (lower(email));

--
-- Name: idx_seats_invite_token_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_seats_invite_token_hash ON public.seats USING btree (invite_token_hash) WHERE (invite_token_hash IS NOT NULL);

--
-- Name: idx_seats_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seats_license ON public.seats USING btree (license_id);

--
-- Name: idx_seats_unique_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_seats_unique_active ON public.seats USING btree (license_id, email) WHERE (removed_at IS NULL);

--
-- Name: idx_seats_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_seats_user ON public.seats USING btree (user_id);

--
-- Name: idx_subscriptions_external; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_subscriptions_external ON public.subscriptions USING btree (payment_provider, external_id);

--
-- Name: idx_subscriptions_license; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_subscriptions_license ON public.subscriptions USING btree (license_id);

--
-- Name: idx_subscriptions_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_subscriptions_user ON public.subscriptions USING btree (user_id);

--
-- Name: idx_usage_license_feature; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_license_feature ON public.usage_events USING btree (license_id, feature, recorded_at);

--
-- Name: idx_usage_recorded; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_usage_recorded ON public.usage_events USING btree (recorded_at);

--
-- Name: idx_users_role; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_role ON public.users USING btree (role);

--
-- Name: idx_webhooks_product; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webhooks_product ON public.webhooks USING btree (product_id);

--
-- Name: licenses licenses_init_updates_until; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER licenses_init_updates_until BEFORE INSERT OR UPDATE OF plan_id ON public.licenses FOR EACH ROW EXECUTE FUNCTION public.licenses_init_updates_until();

--
-- Name: licenses licenses_require_gated_feed; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER licenses_require_gated_feed BEFORE INSERT OR UPDATE OF updates_until, plan_id, product_id ON public.licenses FOR EACH ROW EXECUTE FUNCTION public.licenses_require_gated_feed();

--
-- Name: plans plans_prices_disjoint; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER plans_prices_disjoint BEFORE INSERT OR UPDATE OF stripe_price_id, stripe_renewal_price_id ON public.plans FOR EACH ROW EXECUTE FUNCTION public.plans_prices_disjoint();

--
-- Name: plans plans_record_update_terms; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER plans_record_update_terms AFTER INSERT OR UPDATE OF updates_days, license_type ON public.plans FOR EACH ROW EXECUTE FUNCTION public.plans_record_update_terms();

--
-- Name: plans plans_require_gated_feed; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER plans_require_gated_feed BEFORE INSERT OR UPDATE OF updates_days, stripe_renewal_price_id, product_id ON public.plans FOR EACH ROW EXECUTE FUNCTION public.plans_require_gated_feed();

--
-- Name: products products_keep_feed_gated; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER products_keep_feed_gated BEFORE INSERT OR UPDATE OF feed_license_required, type ON public.products FOR EACH ROW EXECUTE FUNCTION public.products_keep_feed_gated();

--
-- Name: activations activations_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activations
    ADD CONSTRAINT activations_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: addons addons_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.addons
    ADD CONSTRAINT addons_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;

--
-- Name: analytics_snapshots analytics_snapshots_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytics_snapshots
    ADD CONSTRAINT analytics_snapshots_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;

--
-- Name: api_keys api_keys_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;

--
-- Name: entitlements entitlements_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entitlements
    ADD CONSTRAINT entitlements_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE CASCADE;

--
-- Name: floating_sessions floating_sessions_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.floating_sessions
    ADD CONSTRAINT floating_sessions_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: license_addons license_addons_addon_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.license_addons
    ADD CONSTRAINT license_addons_addon_id_fkey FOREIGN KEY (addon_id) REFERENCES public.addons(id) ON DELETE CASCADE;

--
-- Name: license_addons license_addons_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.license_addons
    ADD CONSTRAINT license_addons_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: license_renewals license_renewals_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.license_renewals
    ADD CONSTRAINT license_renewals_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: licenses licenses_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plans(id);

--
-- Name: licenses licenses_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: licenses licenses_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;

--
-- Name: metered_billing metered_billing_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.metered_billing
    ADD CONSTRAINT metered_billing_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: notifications notifications_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: oauth_accounts oauth_accounts_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_accounts
    ADD CONSTRAINT oauth_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

--
-- Name: plan_update_terms plan_update_terms_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plan_update_terms
    ADD CONSTRAINT plan_update_terms_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plans(id) ON DELETE CASCADE;

--
-- Name: plans plans_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT plans_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id);

--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

--
-- Name: release_artifacts release_artifacts_release_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release_artifacts
    ADD CONSTRAINT release_artifacts_release_id_fkey FOREIGN KEY (release_id) REFERENCES public.releases(id) ON DELETE CASCADE;

--
-- Name: release_artifacts release_artifacts_signing_key_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release_artifacts
    ADD CONSTRAINT release_artifacts_signing_key_id_fkey FOREIGN KEY (signing_key_id) REFERENCES public.release_signing_keys(id) ON DELETE SET NULL;

--
-- Name: release_signing_keys release_signing_keys_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.release_signing_keys
    ADD CONSTRAINT release_signing_keys_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;

--
-- Name: releases_legacy releases_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.releases_legacy
    ADD CONSTRAINT releases_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE RESTRICT;

--
-- Name: releases releases_product_id_fkey1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.releases
    ADD CONSTRAINT releases_product_id_fkey1 FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE RESTRICT;

--
-- Name: releases_legacy releases_signing_key_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.releases_legacy
    ADD CONSTRAINT releases_signing_key_id_fkey FOREIGN KEY (signing_key_id) REFERENCES public.release_signing_keys(id) ON DELETE SET NULL;

--
-- Name: renewal_reminders renewal_reminders_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.renewal_reminders
    ADD CONSTRAINT renewal_reminders_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: seats seats_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seats
    ADD CONSTRAINT seats_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: seats seats_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seats
    ADD CONSTRAINT seats_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;

--
-- Name: subscriptions subscriptions_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: payment_transactions payment_transactions_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE SET NULL;

--
-- Name: subscriptions subscriptions_plan_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.plans(id);

--
-- Name: subscriptions subscriptions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;

--
-- Name: usage_counters usage_counters_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_counters
    ADD CONSTRAINT usage_counters_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: usage_events usage_events_license_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usage_events
    ADD CONSTRAINT usage_events_license_id_fkey FOREIGN KEY (license_id) REFERENCES public.licenses(id) ON DELETE CASCADE;

--
-- Name: webhook_deliveries webhook_deliveries_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_deliveries
    ADD CONSTRAINT webhook_deliveries_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) ON DELETE CASCADE;

--
-- Name: webhooks webhooks_product_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE CASCADE;

--
--

