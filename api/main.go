package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/url_shortener/database"
	"github.com/url_shortener/routes"
)

func setupRoutes(app *fiber.App) {
	app.Get("/:url", routes.ResolveURL)
	app.Post("/ap1/v1", routes.ShortenURL)

}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Unable to load environment variables-", err)
	}
	dbURL := fmt.Sprintf("postgres://%s:%s@%s/%s", os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_ADDR"), os.Getenv("DB_NAME"))
	if err := database.ConnectPostgres(dbURL); err != nil {
		log.Fatal(err)

	}

	redisDB := 0
	if err := database.ConnectRedis(os.Getenv("REDIS_ADDR"), os.Getenv("REDIS_PASS"), redisDB); err != nil {
		database.CloseConnections()
		log.Fatal(err)
	}
	defer database.CloseConnections()
	app := fiber.New()
	app.Use(logger.New())

	setupRoutes(app)
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = ":3000"
	}
	log.Fatal(app.Listen(port))

}
