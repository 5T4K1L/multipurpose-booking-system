package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/database"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/migrations"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/server"
	"multipurpose-booking-system/api/internal/service"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Use the repository migration directory by default.
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "../../migrations"
	}

	// Apply pending database migrations before serving requests.
	if err := migrations.Run(databaseURL, migrationsPath); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	// Connect to PostgreSQL after migrations succeed.
	dbPool, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	// Services are ready for later HTTP integration. Session lifetime is absolute.
	authenticationService := service.NewAuthenticationService(repository.NewAuthenticationTransactions(dbPool))
	sessionService, err := service.NewSessionService(repository.NewSessionTransactions(dbPool), 24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	webOrigin := os.Getenv("WEB_ORIGIN")
	if webOrigin == "" {
		webOrigin = "http://localhost:3000"
	}
	registrationService := service.NewRegistrationService(repository.NewRegistrationTransactions(dbPool))
	mux.Handle("/api/v1/auth/register", server.NewRegistrationHandler(registrationService, webOrigin, nil))
	loginHandler, err := server.NewLoginHandler(service.NewLoginService(authenticationService, sessionService), webOrigin, nil)
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/api/v1/auth/login", loginHandler)
	logoutHandler, err := server.NewLogoutHandler(sessionService, webOrigin, nil)
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/api/v1/auth/logout", logoutHandler)
	meHandler, err := server.NewMeHandler(sessionService, service.NewAuthorizationService(repository.NewAuthorizationRepository(db.New(dbPool))), webOrigin, nil)
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/api/v1/auth/me", meHandler)
	oauthConfig, err := auth.LoadEnabledOAuthConfig()
	if err != nil {
		log.Fatal(err)
	}
	apiOrigin := os.Getenv("API_PUBLIC_ORIGIN")
	if apiOrigin == "" {
		apiOrigin = "http://localhost:8080"
	}
	creation := os.Getenv("OAUTH_ALLOW_REGISTRATION")
	if creation != "" && creation != "false" && creation != "true" {
		log.Fatal("OAUTH_ALLOW_REGISTRATION must be true or false")
	}
	oauthRoutes, err := server.NewOAuthRoutes(oauthConfig, service.NewOAuthLoginService(authenticationService, sessionService, creation == "true"), webOrigin, apiOrigin, nil)
	if err != nil {
		log.Fatal(err)
	}
	oauthRoutes.Register(mux)

	// Register the API health endpoint.
	mux.HandleFunc("/health", server.HealthHandler)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Database migrations verified")
	log.Println("PostgreSQL connection verified")
	log.Println("Server is running on http://localhost:8080")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
