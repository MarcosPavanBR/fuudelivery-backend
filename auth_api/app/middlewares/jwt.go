package middlewares

import (
	"errors"
	"github.com/carloshomar/fuudelivery/auth-api/app/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"os"
	"strings"
	"time"
)

func ValidateJWT(c *fiber.Ctx) (*jwt.Token, error) {
	tokenString := c.Get("Authorization")
	if len(tokenString) > 7 {
		tokenString = tokenString[7:]
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface {
	}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil

	})
	if err != nil {
		log.Printf("Error parsing token: %v", err)
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
	}
	if token.Valid {
		return token, nil
	}
	return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
}

func GenerateJWT(user *models.User, establishment *models.Establishment) (string, error) {
	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().UTC().Add(time.Hour).Unix(),
	}
	if establishment != nil {
		claims["establishment_id"] = establishment.ID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func GenerateJWTDeliveryMan(user *models.DeliveryMan) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": "deliveryman",
		"exp":  time.Now().UTC().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func JWTAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid token"})
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token claims"})
	}
	c.Locals("claims", claims)
	c.Locals("user_id", claims["sub"])
	return c.Next()
}
