package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/eazylaykzy/greenlight/internal/data"
	"github.com/eazylaykzy/greenlight/internal/jsonlog"
	"github.com/eazylaykzy/greenlight/internal/mailer"
	_ "github.com/lib/pq"
	"os"
	"strings"
	"sync"
	"time"
)

// Declare a string containing the application version number. Later we'll generate this automatically at build time
const version = "1.0.0"

// Define a config struct to hold all the configuration settings for our application. For now, the only configuration
// settings will be the network port that we want the server to listen on, and the name of the current operating
// environment for the application (development, staging, production, etc.). We will read in these
// configuration settings from command-line flags when the application starts.
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

// Define an application struct to hold the dependencies for our HTTP handlers, helpers, and middleware.
// At the moment this only contains a copy of the config struct and a logger.
type application struct {
	config config
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
	logger *jsonlog.Logger
}

func main() {
	// Declare an instance of the config struct.
	var cfg config

	// Read the value of the port and env command-line flags into the config struct. Default to using the
	// port number 8080 and the environment "development" if no corresponding flags are provided.
	flag.IntVar(&cfg.port, "port", 8080, "API server port")

	// Read DSN (Data Source Name) from the command-line flags into the config struct, or app uses default values
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")

	// Read the connection pool settings from command-line flags into the config struct, or app uses default values
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	// Read config variables for the rate limiter
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	// Read the SMTP server configuration settings into the config struct, using the Mailtrap settings as the default values
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "b006df8803776f", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "7cddd41b44337a", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@adeleke.me>", "SMTP sender")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	// Initialize a new jsonlog.Logger which writes any messages *at or above*
	// the INFO severity level to the standard out stream
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Call the openDB helper function to create the connection pool, passing in the config struct.
	// If this returns an error, we log it and exit the application immediately
	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// Defer a call to db.Close so that the connection pool is closed before the main function exits
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	// Also log a message to say that the connection pool has been successfully established
	logger.PrintInfo("database connection pool established", nil)

	// Declare an instance of the application struct, containing the config struct and the logger.
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}

// openDB function returns a sql.DB connection pool.
func openDB(cfg config) (*sql.DB, error) {
	// Use sql.Open to create an empty connection pool, using the DSN from the config struct
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool. Note,
	// passing a value less than or equal to 0 will mean there is no limit
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool. Note, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Use the time.ParseDuration function to convert the idle timeout duration string to a time.Duration type
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	// Set the maximum idle timeout
	db.SetConnMaxIdleTime(duration)

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use PingContext to establish a new connection to the database, passing in the context we created above as a parameter
	// If the connection couldn't be established successfully within the 5 seconds deadline, then this will return an error
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	// Return the sql.DB connection pool
	return db, nil
}
