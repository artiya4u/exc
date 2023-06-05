package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
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
	for i := 0; i < 6; i++ {
		sb.WriteByte(chars[rand.Intn(len(chars))])
	}
	return sb.String()
}

func (us *URLShortener) shortenURL(w http.ResponseWriter, r *http.Request) {
	longURL := r.FormValue("url")
	if longURL == "" {
		http.Error(w, "URL is missing", http.StatusBadRequest)
		return
	}

	shortURL := us.generateShortURL()
	for {
		_, err := us.redisClient.Get(shortURL).Result()
		if err == redis.Nil {
			break
		}
		shortURL = us.generateShortURL()
	}
	err := us.redisClient.Set(shortURL, longURL, 0).Err()
	if err != nil {
		http.Error(w, "Failed to store URL", http.StatusInternalServerError)
		return
	}

	shortenedURL := fmt.Sprintf("%s/%s", us.baseURL, shortURL)
	fmt.Fprintf(w, shortenedURL)
}

func (us *URLShortener) redirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortURL := vars["shortURL"]

	longURL, err := us.redisClient.Get(shortURL).Result()
	if err != nil {
		if err == redis.Nil {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to retrieve URL", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	redisAddr := getEnv("REDIS_ADDRESS", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	baseURL := getEnv("BASE_URL", "http://localhost:8000")
	urlShortener := NewURLShortener(redisAddr, redisPassword, redisDB, baseURL)

	r := mux.NewRouter()
	r.HandleFunc("/shorten", urlShortener.shortenURL).Methods("POST")
	r.HandleFunc("/{shortURL}", urlShortener.redirectURL).Methods("GET")

	listenAddr := getEnv("LISTEN_ADDRESS", ":8000")
	fmt.Println("URL Shortener is running on", listenAddr, "...")
	log.Fatal(http.ListenAndServe(listenAddr, r))
}
