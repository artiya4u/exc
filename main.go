package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/mr-tron/base58"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	limitFindId = 100
	maxUrl      = 1<<63 - 1
)

type ShortURL struct {
	ID       int64  `json:"id"`
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

var RedisClient *redis.Client
var baseURL string

func main() {
	baseURL = getEnv("BASE_URL", "http://localhost:8000")
	redisURL := getEnv("REDIS_URL", "localhost:6379")
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisURL, // Update with your Redis server address.
		Password: "",       // Set password if required.
		DB:       0,        // Use default Redis database.
	})

	http.HandleFunc("/shorten", shortenURLHandler)
	http.HandleFunc("/", redirectHandler)

	listen := getEnv("LISTEN", ":8000")
	log.Println("Starting server on" + listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}

func shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		URL string `json:"url"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate URL
	_, err = url.ParseRequestURI(requestBody.URL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id := generateID()
	idBytes := make([]byte, 8) // Assuming int64 is 8 bytes
	binary.LittleEndian.PutUint64(idBytes, uint64(id))

	// Encode the ID to Base58.
	shortPath := base58.Encode(idBytes)

	shortURL := ShortURL{
		ID:       id,
		ShortURL: baseURL + "/" + shortPath,
		LongURL:  requestBody.URL,
	}

	err = saveURL(&shortURL)
	if err != nil {
		http.Error(w, "Failed to save URL:"+err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(shortURL)
	if err != nil {
		http.Error(w, "Failed to serialize response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortPath := r.URL.Path[1:]
	if shortPath == "" {
		http.Error(w, "Short path not provided", http.StatusBadRequest)
		return
	}

	longURL, err := getURL(shortPath)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, longURL, http.StatusTemporaryRedirect)
}

func generateID() int64 {
	rand.Seed(time.Now().UnixNano())
	// Random number between 0 and max int64
	return rand.Int63n(maxUrl)
}

func saveURL(shortURL *ShortURL) error {
	key := fmt.Sprintf("url:%d", shortURL.ID)
	found := false
	for i := 0; i < limitFindId; i++ {
		key := fmt.Sprintf("url:%d", shortURL.ID)
		res, _ := RedisClient.Get(key).Result()
		if res != "" {
			shortURL.ID = generateID()
		} else {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("can't find new id")
	}

	_, err := RedisClient.Set(key, shortURL.LongURL, 0).Result()
	if err != nil {
		return err
	}

	return nil
}

func getURL(shortPath string) (string, error) {
	// Decode the Base58 encoded short path to get the ID.
	idBytes, err := base58.Decode(shortPath)
	if err != nil {
		return "", err
	}

	id := binary.LittleEndian.Uint64(idBytes)
	key := fmt.Sprintf("url:%d", id)

	LongURL, err := RedisClient.Get(key).Result()
	if err != nil {
		return "", err
	}

	if len(LongURL) == 0 {
		return "", fmt.Errorf("URL not found")
	}

	return LongURL, nil
}
