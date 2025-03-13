package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

var ctx = context.Background()
var redisClient *redis.Client

func main() {
	// Load .env file if it exists
	godotenv.Load()

	url := os.Getenv("API_URL")
	fmt.Println("URL:", url)
	// Get configuration from environment variables
	redisAddr := os.Getenv("REDIS_ADDR")
	fmt.Println("Redis address: ", redisAddr)

	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	redisAddr = "redis:6379"

	fmt.Println("Redis address: ", redisAddr)
	// Initialize Redis connection
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// Verify Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Redis connection failed: %v", err))
	}

  go func() {
    url := os.Getenv("API_URL")
    apekey := os.Getenv("APE_KEY")
	fmt.Println("URL:", url)
    apiResponse, err := sendAPIRequest(url,apekey)
    fmt.Println("API Response:", apiResponse)
    if err != nil {
      fmt.Println("API request error:", err)
      return
    }
    redisClient.Set(ctx, "type_speed", apiResponse, 0)
  }()
	// Start HTTP server in goroutine
	go func() {
		http.HandleFunc("/", handler)
		fmt.Println("Server starting on :8080")
		http.ListenAndServe(":8080", nil)
	}()

	// Configure cron job
	c := cron.New()
	c.AddFunc("@hourly", func() {
		// Get value from Redis
		val, err := redisClient.Get(ctx, "type_speed").Result()
		if err != nil {
			fmt.Println("Cron error:", err)
			return
		}

		// Send request to API
    url := os.Getenv("API_URL")
    apekey := os.Getenv("APE_KEY")
		apiResponse, err := sendAPIRequest(url,apekey)
		if err != nil {
			fmt.Println("API request error:", err)
			return
		}

		fmt.Printf("Cron executed at %s - Counter: %s - API Response: %s\n",
			time.Now().Format(time.RFC3339), val, apiResponse)
		redisClient.Set(ctx, "type_speed", 0, 0)
	})

	// Initialize counter
	redisClient.Set(ctx, "type_speed", 0, 0)

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
  
  if data, ok := result["data"]; ok {
    if dataMap, ok := data.(map[string]interface{}); ok {
      if wpm, ok := dataMap["wpm"].(float64); ok {
        fmt.Println("WPM:", wpm)
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
