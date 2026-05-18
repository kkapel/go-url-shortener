package audit

import (
	"bytes"
	"encoding/json"
	"go-url-shortener/internal/cookies"
	"go-url-shortener/internal/handler"
	"go-url-shortener/internal/loger"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type AuditFormat struct {
	Ts     int64  `json:"ts"`
	Action string `json:"action"`
	UserID int    `json:"user_id"`
	Url    string `json:"url"`
}

type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	auditResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *auditResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *auditResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

func Audit(auditChan chan<- AuditFormat) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body []byte
			var err error
			responseData := &responseData{
				status: 0,
				size:   0,
			}
			lw := auditResponseWriter{
				ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
				responseData:   responseData,
			}

			if r.Method == http.MethodPost {
				body, err = io.ReadAll(r.Body)
				if err != nil {
					loger.Log.Error("audit.go", zap.String("Function Audit", err.Error()))
				}
				r.Body = io.NopCloser(bytes.NewBuffer(body))
			}

			next.ServeHTTP(&lw, r)

			// Анализируем статус ответа
			if lw.responseData.status >= 200 && lw.responseData.status < 300 {
				// Пишем в канал структуру для отправки в аудит
				auditFormatToChan := AuditFormat{
					Ts: time.Now().Unix(),
					Action: func() string {
						if r.Method == http.MethodPost {
							return "shorten"
						}
						return "follow"
					}(),
					UserID: func() int {
						cookie, err := r.Cookie("access_token")
						if err != nil {
							if err == http.ErrNoCookie {
								return 0
							} else {
								loger.Log.Error("audit.go", zap.String("Function Audit", err.Error()))
								return 0
							}
						}
						id, err := cookies.GetUserID(cookie.Value)
						if err != nil {
							loger.Log.Error("audit.go", zap.String("Function Audit", err.Error()))
							return 0
						}

						return id
					}(),
					Url: func() string {
						// Получаем оригинальный Url
						// Его местонахождение зависит от метода
						// POST / - url в теле запроса
						var longURL string
						if r.Method == http.MethodPost && r.URL.Path == "/" {
							longURL = string(body)
							return longURL
						} else if r.Method == http.MethodPost && r.URL.Path == "/api/shorten" {
							var url handler.URL

							if err := json.Unmarshal(body, &url); err != nil {
								loger.Log.Error("audit.go", zap.String("Function Audit", err.Error()))
								return ""
							}
							longURL = url.URL
						} else if r.Method == http.MethodGet {
							location := w.Header().Get("Location")
							longURL = location
						}

						return longURL

					}(),
				}

				// Отправляем структуру в канал
				select {
				case auditChan <- auditFormatToChan:
				default:
					loger.Log.Error("audit.go", zap.String("Function Audit", "Audit channel is full"))
				}
			}
		})
	}
}

// Функция обработки
// Отправка в файл
// И отправка сообщения на сервер
func ProcessAudit(auditChan <-chan AuditFormat, filePath string, mu *sync.Mutex, auditURL string) {
	var file *os.File
	var err error

	if filePath != "" {
		flag := os.O_APPEND | os.O_WRONLY | os.O_CREATE
		file, err = os.OpenFile(filePath, flag, 0666)

		if err != nil {
			loger.Log.Error("audit.go", zap.String("Function processAudit", err.Error()))
		} else {
			defer file.Close()
		}

	}

	for audit := range auditChan {
		auditJSON, err := json.Marshal(audit)

		if err != nil {
			loger.Log.Error("audit.go", zap.String("Function processAudit", err.Error()))
			continue
		}
		if filePath != "" && file != nil {
			err := func() error {
				var auditJSONFile []byte
				mu.Lock()
				defer mu.Unlock()
				auditJSONFile = append(auditJSON, '\n')
				_, err = file.Write(auditJSONFile)
				return err
			}()
			if err != nil {
				loger.Log.Error("audit.go", zap.String("Function processAudit", err.Error()))
				continue
			}
		}
		// еще делаем отправку на сервер
		if auditURL != "" {
			resp, err := http.Post(auditURL, "application/json", bytes.NewBuffer(auditJSON))
			if err != nil {
				loger.Log.Error("audit.go", zap.String("Function processAudit", err.Error()))
				continue
			}
			resp.Body.Close()

			if resp.StatusCode >= 400 {
				loger.Log.Error("audit.go", zap.String("Function processAudit", http.StatusText(resp.StatusCode)))
			}
		}

	}
}
