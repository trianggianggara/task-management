package middleware

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"task-management/pkg/utils/response"
)

const (
	UserIDKey = "user_id"
	TeamIDKey = "team_id"
)

func Auth(jwtSecret string) echo.MiddlewareFunc {
	secret := []byte(jwtSecret)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return response.Unauthorized("missing authorization header")
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return response.Unauthorized("invalid authorization format")
			}

			token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return secret, nil
			})

			if err != nil || !token.Valid {
				return response.Unauthorized("invalid or expired token")
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return response.Unauthorized("invalid token claims")
			}

			sub, _ := claims["sub"].(string)
			teamID, _ := claims["team_id"].(string)

			if sub == "" {
				return response.Unauthorized("invalid token subject")
			}

			c.Set(UserIDKey, sub)
			c.Set(TeamIDKey, teamID)

			return next(c)
		}
	}
}

func GetUserID(c echo.Context) string {
	id, _ := c.Get(UserIDKey).(string)
	return id
}

func GetTeamID(c echo.Context) string {
	id, _ := c.Get(TeamIDKey).(string)
	return id
}
