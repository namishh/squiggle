create extension if not exists pgcrypto;
create extension if not exists pg_trgm;

create table if not exists entries (
    id uuid primary key default gen_random_uuid(),
    name varchar(100) not null,
    message text not null,
    site text,
    email varchar(255),
    status varchar(20) not null default 'hidden',
    ip_hash varchar(255),
    user_agent text,
    sentiment_score real default 0.2,
    created_at timestamptz not null default now(),
    search_vector   tsvector generated always as (
                        to_tsvector('english', coalesce(name,'') || ' ' || coalesce(message,''))
                    ) stored
);

create table if not exists defaulters (
    ip_hash             varchar(255) primary key,
    email               varchar(255),
    low_sentiment_count integer not null default 0,
    banned              boolean not null default false,
    last_offense_at     timestamptz not null default now()
);

create index if not exists idx_entries_search on entries using gin (search_vector);
create index if not exists idx_entries_created_at on entries (created_at desc);
create index if not exists idx_entries_message_trgm on entries using gin (message gin_trgm_ops);
create index if not exists idx_entries_visible on entries (created_at desc) where status = 'visible';
create index if not exists idx_entries_hidden on entries (created_at desc) where status = 'hidden';
create index if not exists idx_entries_iphash_created on entries (ip_hash, created_at);
