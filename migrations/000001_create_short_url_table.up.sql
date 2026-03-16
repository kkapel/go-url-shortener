-- Создание таблицы short_url
create table if not exists short_url
(
id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
short_link VARCHAR(100) UNIQUE,
long_link VARCHAR(500) UNIQUE,
user_id int,
deleted_flag boolean,
change_time timestamp
);

comment on column short_url.id is 'ID записи';
comment on column short_url.short_link is 'Короткий URL';
comment on column short_url.long_link is 'Длинный URL';
comment on column short_url.deleted_flag is 'Флаг удаленного сокращенного URL';
comment on column short_url.change_time is 'Время занесения/изменения значения';