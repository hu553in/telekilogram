create table if not exists user_settings (
  user_id integer primary key,
  auto_digest_hour_utc integer not null default 0 check (
    auto_digest_hour_utc >= 0
    and auto_digest_hour_utc < 24
  )
);

create index if not exists idx_user_settings_auto_digest_hour_utc on user_settings (auto_digest_hour_utc);
