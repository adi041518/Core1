package server

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/adi041518/Core1/config/db"
	"github.com/adi041518/Core1/config/redis"
	"github.com/gin-gonic/gin"
)

var ctx context.Context

type Options struct {
	CacheEnabled        bool
	MongoEnabled        bool
	WebServerEnabled    bool
	WebServerPort       string
	JobsEnabled         bool
	JobsHandler         func()
	WebServerPreHandler func(*gin.Engine)
	MigrationHandler    func()
	MigrationEnabled    bool
}

func GetDefaultOptions() Options {
	return Options{
		CacheEnabled:        true,
		MongoEnabled:        true,
		WebServerEnabled:    true,
		WebServerPort:       "8080",
		WebServerPreHandler: nil,
		JobsEnabled:         true,
		JobsHandler:         nil,
		MigrationHandler:    nil,
		MigrationEnabled:    true,
	}
}
func Start(options Options) {

	ctx = context.TODO()
	if options.MongoEnabled {
		db.ConnectDB()
	}
	if options.CacheEnabled {
		redis.ConnectRedis()
	}
	if options.JobsHandler != nil && options.JobsEnabled {
		options.JobsHandler()
	}
	if options.MigrationHandler != nil && options.MigrationEnabled {
		options.MigrationHandler()
	}
	initServer(&options)

}
func initServer(options *Options) {
	if !options.WebServerEnabled {
		log.Println("server: not enabled")
		return
	}
	server := gin.Default()
	if options.WebServerPreHandler != nil {
		options.WebServerPreHandler(server)
	}
	// Use the default value of "8080" if WebServerPort is empty
	port := options.WebServerPort
	customPort := os.Getenv("web_server_port")
	if customPort != "" {
		port = customPort
	}
	if port == "" {
		port = "8080" // Default port if not provided
	}
	if os.Getenv("ssl_enabled") == "Y" {
		slog.Info("server starting on " + port)
		if err := server.RunTLS(":"+port, "server.crt", "server.key"); err != nil {
			log.Fatal(err)
		}
	} else {
		slog.Info("server starting on " + port)
		if err := server.Run(":0.0.0.0" + port); err != nil {
			log.Fatal(err)
		}
	}
}
