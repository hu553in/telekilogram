create table if not exists feeds (
  id integer primary key autoincrement,
  user_id integer not null,
  url text not null,
  title text not null,
  unique (user_id, url)
);
