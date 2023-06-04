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
	"time"
)

const (
	baseURL  = "http://localhost:8000/"
	redisURL = "localhost:6379"
)

type ShortURL struct {
	ID        int64  `json:"id"`
	ShortPath string `json:"short_path"`
	LongURL   string `json:"long_url"`
}

var RedisClient *redis.Client

func main() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisURL, // Update with your Redis server address.
		Password: "",       // Set password if required.
		DB:       0,        // Use default Redis database.
	})

	http.HandleFunc("/shorten", shortenURLHandler)
	http.HandleFunc("/", redirectHandler)

	log.Println("Starting server on http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	longURL := r.FormValue("url")
	if longURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	id := generateID()
	idBytes := make([]byte, 8) // Assuming int64 is 8 bytes
	binary.LittleEndian.PutUint64(idBytes, uint64(id))

	// Encode the ID to Base58.
	shortPath := base58.Encode(idBytes)

	shortURL := ShortURL{
		ID:        id,
		ShortPath: baseURL + "/" + shortPath,
		LongURL:   longURL,
	}

	err := saveURL(shortURL)
	if err != nil {
		http.Error(w, "Failed to save URL", http.StatusInternalServerError)
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
	return rand.Int63n(1000000)
}

func saveURL(shortURL ShortURL) error {
	key := fmt.Sprintf("url:%d", shortURL.ID)

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
	println(key)

	LongURL, err := RedisClient.Get(key).Result()
	if err != nil {
		return "", err
	}

	if len(LongURL) == 0 {
		return "", fmt.Errorf("URL not found")
	}

	return LongURL, nil
}
