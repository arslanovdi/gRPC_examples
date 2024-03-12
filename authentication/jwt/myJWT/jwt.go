package myJWT

import (
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"time"
)

var (
	ErrInvalidToken = status.Errorf(codes.Unauthenticated, "invalid credentials")
)

const secretKey = "секретный ключ"

// Payload кастомная реализация
type Payload struct {
	User string `json:"user,omitempty"`
	jwt.RegisteredClaims
}

// NewPayload интегрируем данные в jwt
func NewPayload(user string, exp time.Duration) jwt.Claims {
	payload := Payload{
		user,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "test",
			Subject:   "somebody",
			ID:        "1",
			Audience:  []string{"somebody_else"},
		},
	}
	return &payload
}

// NewToken собирает jwt токен
func NewToken(user string, exp time.Duration) (string, error) {

	p := NewPayload(user, exp) // собираем Payload данные

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, p) // создаем JWT токен с Payload

	tokenString, err := token.SignedString([]byte(secretKey)) // подписывание токена

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// VerifyToken валидация jwt токена и выгрузка данных из него
func VerifyToken(token string) (bool, error) {

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		return false, err
	} else if payload, ok := jwtToken.Claims.(*Payload); ok {
		slog.Info("Payload", slog.Any("user", payload.User), slog.Any("IssuedAt", payload.IssuedAt))
	} else {
		slog.Error("Payload error")
	}

	return jwtToken.Valid, nil
}
