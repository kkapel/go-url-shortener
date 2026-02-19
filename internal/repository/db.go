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

func InitDB(dbConnect string) *DB {
	db, err := sql.Open("pgx", dbConnect)
	if err != nil {
		// убрать панику
		panic(err)
	}

	return &DB{
		db: db,
	}
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
func (db *DB) GetUrlFromDB(ctx context.Context, inputURL string, URLType string) (string, error) {
	var sql_str string
	switch URLType {
	case "short":
		// Получаем LongURL
		sql_str = "select long_link from short_url where short_link = ?"
	case "long":
		// Получаем short_url
		sql_str = "select short_link from short_url where long_link = ?"
	}

	var url_db sql.NullString
	row := db.db.QueryRowContext(ctx, sql_str, inputURL)

	err := row.Scan(&url_db)

	if err != nil {
		return "", err
	}

	if url_db.Valid {
		return url_db.String, nil
	}

	return "", nil

}

// Функция записи ссылок в БД
func (db *DB) InsertIntoDB(ctx context.Context, inputURL string, URLType string) error {
	var sql_str string
	switch URLType {
	case "short":
		sql_str = "insert into short_url (short_link) values (?)"
	case "long":
		sql_str = "insert into short_url (long_link) values (?)"
	}

	_, err := db.db.ExecContext(ctx, sql_str, inputURL)

	if err != nil {
		return err
	}

	return nil
}
