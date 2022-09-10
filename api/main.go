package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Askia/redis-poc/internal/data"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type App struct {
	rdb *redis.Client
	exp time.Duration
}

// cors, CORS handler
func cors(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "origin, content-type, accept, authorization")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}

// NewApp, application settings
func NewApp() *App {
	h := os.Getenv("REDIS_HOSTNAME")
	if h == "" {
		h = "localhost"
	}
	p := os.Getenv("REDIS_PORT")
	if p == "" {
		p = "6379"
	}
	a := fmt.Sprintf("%s:%s", h, p)
	rdb := redis.NewClient(&redis.Options{
		Addr:     a,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result() // test redis connection
	if err != nil {
		log.Printf("cannot connect to Redis Server %q\n", a)
		panic("cannot connect to Redis Server")
	}
	log.Printf("🤗 Connected with Redis Server %q\n", a)

	return &App{rdb, 60 * time.Second} //@TODO: inject duration from var env
}

func (app App) save(c *gin.Context) {
	k := c.Param("key")

	var m data.Message
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m.Timestamp = time.Now().UnixMilli()
	bytes, err := json.Marshal(m) // marshal json into bytes[]
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = app.rdb.Set(c, k, bytes, app.exp).Err() // set data + expiration
	if err != nil {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"key":     k,
		"message": m})
}

func (app App) find(c *gin.Context) {
	k := c.Param("key")

	v, err := app.rdb.Get(c, k).Result()
	if errors.Is(err, redis.Nil) {
		c.JSON(http.StatusNotFound, gin.H{"error": "key does not exist", "key": k})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		var m data.Message
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"key":     k,
			"message": m})
	}
}

func setupRouter(app App) *gin.Engine {
	r := gin.Default()
	// r.MaxMultipartMemory = 16 << 20 // 16 MiB

	r.Use(cors)
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/:key", app.save)
	r.GET("/:key", app.find)
	return r
}

func main() {
	app := *NewApp()
	r := setupRouter(app)
	defer app.rdb.Close()

	var addr string
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	} else {
		addr = ":8080" // default port
	}

	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   20 * time.Second,
		IdleTimeout:    time.Minute,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		s := <-quit
		log.Printf("🚨 Shutdown signal \"%v\" received\n", s)

		log.Printf("🚦 Here we go for a graceful Shutdown...\n")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("⚠️ HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("✅ Listening and serving HTTP on %s\n", addr)
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("👹 HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
	log.Printf("😴 Server stopped")
}
