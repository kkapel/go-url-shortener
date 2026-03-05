-- Создание таблицы users_token
create table if not exists users_token
(
id serial primary key,
accessToken VARCHAR(100) UNIQUE
);

comment on column users_token.id is 'ID пользователя';
comment on column users_token.accessToken is 'Токен пользователя';