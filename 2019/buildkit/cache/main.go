package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"bitbucket.org/okteto/okteto/backend/api"
	"bitbucket.org/okteto/okteto/backend/app"
	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/events"
	"bitbucket.org/okteto/okteto/backend/logger"
	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/store"

	"github.com/golang-migrate/migrate/database"
	"github.com/golang-migrate/migrate/database/postgres"
	pq "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/jinzhu/gorm"
)

func main() {
	config.LoadConfig()

	port := os.Getenv("PORT")
	if port == "" {
		port = "80000"
	}

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "8001"
	}

	gormDB, listener := startDatabaseConnections()

	server := &app.Server{
		Hub:         events.NewHub(),
		Listener:    listener,
		DNSProvider: getDNSProvider(),
		Email:       getEmailProvider(),
		DB:          gormDB,
	}

	server.API = &http.Server{
		Handler:      api.Init(server),
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	server.Metrics = &http.Server{
		Handler:      prometheus.Handler(),
		Addr:         fmt.Sprintf("0.0.0.0:%s", metricsPort),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	server.Start()
	sigint := make(chan os.Signal)

	// interrupt signal sent from terminal
	signal.Notify(sigint, os.Interrupt)
	// sigterm signal sent from kubernetes
	signal.Notify(sigint, syscall.SIGTERM)
	logger.Info("started api service on %s and metrics service on %s", server.API.Addr, server.Metrics.Addr)

	s := <-sigint
	logger.Info("received termination signal: %s", s.String())
	server.Stop()
	logger.Info("stopped api service")
}

func errorReporter(ev pq.ListenerEventType, err error) {
	if err != nil {
		logger.Error(errors.Wrap(err, "postgres listener returned an error"))
	}
}

func openDatabase() (*gorm.DB, error) {
	driver, dbinfo := config.GetDBConnectionString()
	var gormdb *gorm.DB
	for i := 30; i > 0; i-- {
		var err error
		gormdb, err = gorm.Open(driver, dbinfo)
		if err == nil {
			break
		}

		if i == 0 {
			return nil, errors.Wrap(err, "Failed to open the DB after 10 attempts")
		}

		logger.Info(fmt.Sprintf("Failed to ping the DB, retrying: %s", err))
		time.Sleep(time.Second)
	}

	gormdb.LogMode(config.GetSQLTrace())
	gorm.NowFunc = func() time.Time {
		return time.Now().UTC()
	}

	gormdb.DB().SetMaxIdleConns(10)
	gormdb.DB().SetMaxOpenConns(100)
	gormdb.DB().SetConnMaxLifetime(time.Duration(60) * time.Minute)
	return gormdb, nil
}

func openListener() *pq.Listener {
	_, dbinfo := config.GetDBConnectionString()
	listener := pq.NewListener(dbinfo, 10*time.Second, time.Minute, errorReporter)
	err := listener.Listen("events")
	if err != nil {
		logger.Fatal(errors.Wrap(err, "Failed to start the postgres event listener"))
	}

	return listener
}

func getDriver(db *gorm.DB) database.Driver {
	driver, err := postgres.WithInstance(db.DB(), &postgres.Config{})
	if err != nil {
		logger.Fatal(errors.Wrap(err, "Failed to create a postgres driver"))
	}

	return driver
}

func getMigrationPath() string {
	ex, err := os.Executable()
	if err != nil {
		logger.Fatal(errors.Wrap(err, "Failed to determine the local path"))
	}

	exPath := filepath.Dir(ex)
	return fmt.Sprintf("file://%s/migrations", exPath)
}

func startDatabaseConnections() (*gorm.DB, *pq.Listener) {
	gormdb, err := openDatabase()
	if err != nil {
		log.Fatal(err.Error())
	}

	listener := openListener()
	migrationsPath := getMigrationPath()
	driver := getDriver(gormdb)
	err = store.InitSQLStore(gormdb, driver, migrationsPath)

	if err != nil {
		logger.Fatal(err)
	}

	return gormdb, listener
}

func getDNSProvider() *model.DNSProvider {
	if !config.IsDNSConfigured() {
		return nil
	}
	hostedZone, accessKey, secretKey := config.GetDNSCredentials()
	dnsProvider := &model.DNSProvider{
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		HostedZone: hostedZone,
	}
	err := dnsProvider.Validate()
	if err != nil {
		logger.Fatal(errors.Wrap(err, "Invalid DNS credentials"))
	}
	return dnsProvider
}

func getEmailProvider() *app.EmailProvider {
	var sender app.EmailSender
	domain, apiKey := config.GetMailgunCredentials()
	if domain != "" && apiKey != "" {
		log.Printf("using mailgun to send email notifications")
		sender = app.NewMailgunSender(apiKey, domain)
	} else {
		sender = &app.NoopMail{}
	}

	return app.NewMail(config.GetNotificationEmail(), sender)
}
