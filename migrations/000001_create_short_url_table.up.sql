-- Создание таблицы short_url
create table if not exists short_url
(
id serial primary key,
short_link VARCHAR(100) UNIQUE,
long_link VARCHAR(500) UNIQUE
);

comment on column short_url.id is 'ID записи';
comment on column short_url.short_link is 'Короткий URL';
comment on column short_url.long_link is 'Длинный URL';