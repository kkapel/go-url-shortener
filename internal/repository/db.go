package repository

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	db *sql.DB
}

type UniqueViolationError struct {
	LongURL string
	Err     error
}

func (e *UniqueViolationError) Error() string {
	return fmt.Sprintf("url %s already exists, original error: %v", e.LongURL, e.Err)
}

func (e *UniqueViolationError) Unwrap() error {
	return e.Err
}

func NewUniqueViolationError(longURL string, err error) error {
	return &UniqueViolationError{
		LongURL: longURL,
		Err:     err,
	}
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
// Для Post-запросов при нахождении короткой ссылки возвращаем http status 409 Conflict
func (db *DB) GetURLFromDB(ctx context.Context, inputURL string, URLType string, httpMethod string) (string, error) {
	var sqlStr string
	switch URLType {
	case "long":
		// Получаем LongURL
		sqlStr = "select long_link from short_url where short_link = $1"
	case "short":
		// Получаем short_url
		sqlStr = "select short_link from short_url where long_link = $1"
	}

	var urlDB sql.NullString
	row := db.db.QueryRowContext(ctx, sqlStr, inputURL)

	err := row.Scan(&urlDB)

	// Проверка кейса http status 409 Conflict
	// При попытке пользователя сократить уже имеющийся в базе URL через хендлеры POST / и POST /api/shorten сервис должен вернуть HTTP-статус 409 Conflict,
	// а в теле ответа — уже имеющийся сокращённый URL в правильном для хендлера формате.

	if httpMethod == http.MethodPost && err != sql.ErrNoRows && URLType == "short" {
		return "", NewUniqueViolationError(urlDB.String, fmt.Errorf("UniqueViolationError"))
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	if urlDB.Valid {
		return urlDB.String, nil
	}

	return "", nil

}

// Функция записи ссылок в БД
func (db *DB) InsertIntoDB(ctx context.Context, shortURL string, longURL string) error {
	sqlStr := "insert into short_url (short_link, long_link) values ($1, $2)"

	_, err := db.db.ExecContext(ctx, sqlStr, shortURL, longURL)

	if err != nil {
		return err
	}

	return nil
}
