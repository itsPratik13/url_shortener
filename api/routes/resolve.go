package routes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/url_shortener/database"
)

var (
	URL_NOT_FOUND = errors.New("URL not found")
	URL_EXPIRED   = errors.New("URL is expired")
)

func GetOriginalURL(ctx context.Context, shortUrl string) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	//check if redis has it
	cacheKey := "url:" + shortUrl
	originalURL, err := database.Redis.Get(ctx, cacheKey).Result()
	if err == nil {
		return originalURL, nil
	}
	if err != redis.Nil {
		fmt.Println("Error in Redis: ", err)
	}

	//if not found in redis,  check in db ,if found put in redis and if not found error
	var expired bool
	var cacheTTL float64
	err = database.DB.QueryRow(ctx,
		`select original_url, (expires_at is not null and expires_at<=now()),
		case
		    when expires_at is null  then 86400
			else greatest(0,extract(epoch from(expires_at-now())):: double precision)
		end
		from public.urls
		where short_code=$1	
		`, shortUrl).Scan(&originalURL, &expired, &cacheTTL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", URL_NOT_FOUND
		}
		return "", fmt.Errorf("Database Error: %w", err)

	}
	//check expiration
	if expired {
		return "", URL_EXPIRED
	}
	ttl := time.Duration(cacheTTL * float64(time.Second))
	if ttl > 0 {
		err = database.Redis.Set(ctx, "url:"+shortUrl, originalURL, ttl).Err()

		if err != nil {
			fmt.Println("Redis cache error", err)
		}
	}
	//return orignal key
	return originalURL, nil

}

func ResolveURL(ctx *fiber.Ctx) error {
	shortUrl := ctx.Params("url")
	if shortUrl==""{
		return ctx.Status(400).JSON(fiber.Map{"error":"Short URL is required"})
	}
	originalURL, err := GetOriginalURL(ctx.UserContext(), shortUrl)
	if err != nil {
		if errors.Is(err, URL_NOT_FOUND) {
			return ctx.Status(404).JSON(fiber.Map{"error": "Url not found"})
		}
		if errors.Is(err, URL_EXPIRED) {
			return ctx.Status(410).JSON(fiber.Map{"error": "Url expired"})
		}
		return ctx.Status(500).JSON(fiber.Map{"error":"Internal Server error"})
	}
	return ctx.Redirect(originalURL,302)
}
