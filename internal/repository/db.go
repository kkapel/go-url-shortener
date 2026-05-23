package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-url-shortener/internal/loger"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	"go.uber.org/zap"
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

	databaseInstance := &DB{db: db}

	if err = databaseInstance.CheckConnect(); err != nil {
		return nil, err
	}

	if err = migrateDB(dbConnect); err != nil {
		return nil, err
	}
	return databaseInstance, nil
}

func migrateDB(dbConnect string) error {
	m, err := migrate.New("file://migrations", dbConnect)
	if err != nil {
		return err
	}
	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (db *DB) CheckConnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.db.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (db *DB) Close() error {
	err := db.db.Close()
	if err != nil {
		return err
	}
	return nil
}

// Функция получения URL из Базы Данных
// При значении NULL возвращается пустая строка
// Для Post-запросов при нахождении короткой ссылки возвращаем http status 409 Conflict
func (db *DB) GetURLFromDB(ctx context.Context, inputURL string, URLType string, action string) (string, error) {
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

	if action == "Post" && err != sql.ErrNoRows && URLType == "short" {
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
func (db *DB) InsertIntoDB(ctx context.Context, shortURL string, longURL string, userID int) error {
	sqlStr := "insert into short_url (short_link, long_link, user_id, change_time) values ($1, $2, $3, $4)"

	loger.Log.Info("DB Exec",
		zap.String("query", sqlStr),
		zap.String("short", shortURL),
		zap.String("long", longURL),
		zap.Int("user_id", userID),
	)

	var userIDInsert sql.NullInt32
	if userID != 0 {
		userIDInsert = sql.NullInt32{Int32: int32(userID), Valid: true}
	} else {
		userIDInsert = sql.NullInt32{Valid: false} //null value
	}

	_, err := db.db.ExecContext(ctx, sqlStr, shortURL, longURL, userIDInsert, time.Now())

	if err != nil {
		return err
	}

	return nil
}

func (db *DB) GetNextUserID(ctx context.Context) (int, error) {

	if db == nil {
		return 0, nil
	}
	sqlStr := "SELECT nextval('users_token_id_seq')"

	row := db.db.QueryRowContext(ctx, sqlStr)

	var urlDB sql.NullInt32
	err := row.Scan(&urlDB)

	if err != nil {
		return 0, err
	}

	if urlDB.Valid {
		return int(urlDB.Int32), nil
	}

	return 0, nil
}

// Функция записи токена в БД
func (db *DB) InsertUserID(ctx context.Context, id int, accessToken string) error {

	if db == nil {
		return nil
	}

	sqlStr := `
    INSERT INTO users_token (id, accessToken) 
    VALUES ($1, $2) 
    ON CONFLICT (id) DO UPDATE SET accessToken = EXCLUDED.accessToken
`
	_, err := db.db.ExecContext(ctx, sqlStr, id, accessToken)

	if err != nil {
		return err
	}

	return nil

}

func (db *DB) GetURLsByUserID(ctx context.Context, userID int) (map[string]string, error) {

	if db == nil {
		return nil, nil
	}

	sqlStr := "select short_link, long_link from short_url where user_id = $1"
	rows, err := db.db.QueryContext(ctx, sqlStr, userID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Строка не найдена
		}
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)

	for rows.Next() {
		var shortURL, longURL string
		if err = rows.Scan(&shortURL, &longURL); err != nil {
			return nil, err
		}
		result[shortURL] = longURL
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil

}

func (db *DB) SetDeletedFlag(ctx context.Context, ids []string, id int) error {
	if db == nil {
		return nil
	}

	sqlStr := "update short_url set deleted_flag = true where user_id = $1 and short_link = ANY($2)"

	_, err := db.db.ExecContext(ctx, sqlStr, id, pq.Array(ids))

	if err != nil {
		return err
	}

	return nil

}

func (db *DB) CheckFlagDeleteExists(ctx context.Context, shortLink string) (bool, error) {
	if db == nil {
		return false, nil
	}

	var exists bool
	sqlStr := "SELECT EXISTS(SELECT 1 FROM short_url WHERE short_link = $1 AND deleted_flag = true)"

	err := db.db.QueryRowContext(ctx, sqlStr, shortLink).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil

}
