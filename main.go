package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

var ctx = context.Background()
var redisClient *redis.Client

type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("\033[32m[INFO]\033[0m "+msg, args...)  // Green
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("\033[31m[ERROR]\033[0m "+msg, args...) // Red
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	log.Printf("\033[36m[DEBUG]\033[0m "+msg, args...) // Cyan
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("\033[33m[WARN]\033[0m "+msg, args...)  // Yellow
}

var logger Logger = &defaultLogger{}

func main() {
	// Load .env file if it exists
	godotenv.Load()

	// Get configuration from environment variables
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	logger.Info("Redis address config: %s", redisAddr)

	// Initialize Redis connection
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	logger.Info("Connecting to Redis at %s", redisAddr)

	// Verify Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Redis connection failed: %v", err))
	}

	// Start HTTP server in goroutine
	go func() {
		http.HandleFunc("/", handler)
		logger.Warn("Starting server on :8080")
		http.ListenAndServe(":8080", nil)
	}()

	// Create cron job
	c := cron.New()
	apekey := os.Getenv("APE_KEY")
	url := os.Getenv("API_URL")
	
	// Initialize counter
	redisClient.Set(ctx, "type_speed", 0, 0)
	
	// Start cron job
	c.AddFunc("@every 5s", func() {
		
		// Send request to API
		apiResponse, err := sendAPIRequest(url,apekey)

		redisClient.Set(ctx, "type_speed",apiResponse, 0)
		if err != nil {
			logger.Error("API request error:", err)
			return
		}

		logger.Info("Cron executed set val: %s\n",apiResponse)
	})


	c.Start()
	select {} // Block main goroutine
}

func sendAPIRequest(url string, key string) (string, error) {
  req, err := http.NewRequest("GET",url, nil)
  if err != nil {
    return "Unable to create Request", err
  }
  req.Header.Add("Authorization", "ApeKey " + key)
  client := &http.Client{}
  resp, err := client.Do(req)

  if err != nil {
    return "Request Failed", err
  }
  defer resp.Body.Close()

  body, err := io.ReadAll(resp.Body)
  if err != nil {
    return "", err
  }
  
  var result map[string]interface{}
  if err := json.Unmarshal(body, &result); err != nil {
    return "", err
  }
  
  // Log the complete response data using your custom logger

  
  if data, ok := result["data"]; ok {
    if dataMap, ok := data.(map[string]interface{}); ok {
      // Log the data portion specifically

      
      if wpm, ok := dataMap["wpm"].(float64); ok {
        return fmt.Sprintf("%.2f", wpm), nil
      }
    }
    return fmt.Sprintf("%v", data), nil
  }
  return "", fmt.Errorf("no data field in response")
}

func handler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
    
    // Handle preflight OPTIONS request
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")

    count, err := redisClient.Get(ctx, "type_speed").Result()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"wpm": count})
}
