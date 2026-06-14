package testmodel

import (
	"go-url-shortener/internal/pool"
	"testing"
)

func TestPoolReset(t *testing.T) {
	// Создаем пул для типа User1
	p := pool.New(func() *User1 {
		return &User1{}
	})

	// Получаем объект из пула
	u := p.Get()
	// Заполняем поля структуры
	u.ID = 123
	u.Name = "Greg"

	// Возвращаем объект в пул
	// При возвращении объекта в пул, его поля должны быть сброшены до нулевых значений
	p.Put(u)

	// Получаем объект из пула снова
	u2 := p.Get()

	// Проверяем, что поля сброшены до нулевых значений
	if u2.Name != "" || u2.ID != 0 {
		t.Fatalf("expected empty Name and ID after Reset, got %q and %d", u2.Name, u2.ID)
	}
}
