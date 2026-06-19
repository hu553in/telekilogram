-- name: AddOrIgnoreFeed :exec
insert or ignore into
    feeds (user_id, url, title)
values
    (?, ?, ?);

-- name: UpdateFeedTitle :exec
update feeds
set
    title = ?
where
    id = ?;

-- name: RemoveFeed :exec
delete from feeds
where
    id = ?;

-- name: GetUserFeeds :many
select
    id,
    url,
    title
from
    feeds
where
    user_id = ?;

-- name: GetHourFeedsMidnightUTC :many
select
    f.id,
    f.user_id,
    f.url,
    f.title
from
    feeds as f
    left join user_settings as us on us.user_id = f.user_id
where
    us.user_id is null
    or us.auto_digest_hour_utc = ?;

-- name: GetHourFeeds :many
select
    f.id,
    f.user_id,
    f.url,
    f.title
from
    feeds as f
    left join user_settings as us on us.user_id = f.user_id
where
    us.auto_digest_hour_utc = ?;

-- name: GetUserSettings :one
select
    user_id,
    auto_digest_hour_utc
from
    user_settings
where
    user_id = ?;

-- name: UpsertUserSettings :exec
insert into
    user_settings (user_id, auto_digest_hour_utc)
values
    (?, ?)
on conflict (user_id) do update
set
    auto_digest_hour_utc = excluded.auto_digest_hour_utc;
