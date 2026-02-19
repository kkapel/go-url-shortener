package repository

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	db *sql.DB
}

func InitDB(dbConnect string) (*DB, error) {
	if dbConnect == "" {
		return nil, nil
	}
	db, err := sql.Open("pgx", dbConnect)
	if err != nil {
		return nil, err
	}

	database := &DB{db: db}

	if err := database.CheckConnect(); err != nil {
		return nil, err
	}

	query :=
		`create table IF NOT EXISTS short_url
		(
		id serial primary key,
		short_link VARCHAR(100) UNIQUE,
		long_link VARCHAR(500) UNIQUE
		);

		comment on column short_url.id is 'ID записи';
		comment on column short_url.short_link is 'Короткий URL';
		comment on column short_url.long_link is 'Длинный URL';`

	_, err = db.Exec(query)

	if err != nil {
		return nil, err
	}

	return database, nil
}

func (db *DB) CheckConnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.db.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (db *DB) Close() error {
	db.db.Close()
	return nil
}

// Функция получения URL из Базы Данных
// При значении NULL возвращается пустая строка
func (db *DB) GetURLFromDB(ctx context.Context, inputURL string, URLType string) (string, error) {
	var sqlStr string
	switch URLType {
	case "short":
		// Получаем LongURL
		sqlStr = "select long_link from short_url where short_link = $1"
	case "long":
		// Получаем short_url
		sqlStr = "select short_link from short_url where long_link = $1"
	}

	var urlDB sql.NullString
	row := db.db.QueryRowContext(ctx, sqlStr, inputURL)

	err := row.Scan(&urlDB)

	if err != nil {
		return "", err
	}

	if urlDB.Valid {
		return urlDB.String, nil
	}

	return "", nil

}

// Функция записи ссылок в БД
func (db *DB) InsertIntoDB(ctx context.Context, shortURL string, longURL string) error {
	var sqlStr string
	sqlStr = "insert into short_url (short_link, long_link) values ($1, $2)"

	_, err := db.db.ExecContext(ctx, sqlStr, shortURL, longURL)

	if err != nil {
		return err
	}

	return nil
}
