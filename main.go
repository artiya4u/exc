package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
)

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

type URLShortener struct {
	redisClient *redis.Client
	baseURL     string
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortenedURL string `json:"short_url"`
}

func NewURLShortener(redisAddr, redisPassword string, redisDB int, baseURL string) *URLShortener {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	return &URLShortener{
		redisClient: redisClient,
		baseURL:     baseURL,
	}
}

func (us *URLShortener) generateShortURL() string {
	chars := "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ123456789"
	var sb strings.Builder
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 4; i++ {
		sb.WriteByte(chars[rand.Intn(len(chars))])
	}
	return sb.String()
}

func (us *URLShortener) shortenURLPost(c echo.Context) error {
	req := new(ShortenRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid request")
	}
	return us.shortenURL(req.URL, c)
}

func (us *URLShortener) shortenURLGet(c echo.Context) error {
	URL := c.Param("URL")
	return us.shortenURL(URL, c)
}

func (us *URLShortener) shortenURL(longURL string, c echo.Context) error {
	if longURL == "" {
		return c.JSON(http.StatusBadRequest, "URL is missing")
	}
	shortURL := us.generateShortURL()
	for {
		set, err := us.redisClient.SetNX(c.Request().Context(), shortURL, longURL, 0).Result()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "Failed to store URL")
		}
		if set {
			break
		} else {
			shortURL = us.generateShortURL()
		}
	}

	shortenedURL := fmt.Sprintf("%s/%s", us.baseURL, shortURL)
	res := ShortenResponse{ShortenedURL: shortenedURL}
	return c.JSON(http.StatusOK, res)
}

func (us *URLShortener) redirectURL(c echo.Context) error {
	shortURL := c.Param("shortURL")

	longURL, err := us.redisClient.Get(c.Request().Context(), shortURL).Result()
	if err != nil {
		if err == redis.Nil {
			return c.JSON(http.StatusNotFound, "Short URL not found")
		} else {
			return c.JSON(http.StatusInternalServerError, "Failed to retrieve URL")
		}
	}

	return c.Redirect(http.StatusFound, longURL)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	redisAddr := getEnv("REDIS_ADDRESS", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	baseURL := getEnv("BASE_URL", "http://localhost:8000")
	urlShortener := NewURLShortener(redisAddr, redisPassword, redisDB, baseURL)

	e := echo.New()

	e.POST("/shorten", urlShortener.shortenURLPost)
	e.GET("/shorten/:URL", urlShortener.shortenURLGet)
	e.GET("/:shortURL", urlShortener.redirectURL)

	listenAddr := getEnv("LISTEN_ADDRESS", ":8000")
	log.Fatal(e.Start(listenAddr))
}
