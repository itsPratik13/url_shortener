package routes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/url_shortener/database"
)

var (
	URL_NOT_FOUND = errors.New("URL not found")
	URL_EXPIRED   = errors.New("URL is expired")
)

func ResolveURL(shortUrl string) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	row := database.DB.QueryRow(ctx,
		`select original_url, (expires_at is not null and expires_at<=now()),
		case
		    when expires_at is null  then 86400
			else greatest(0,extract(epoch from(expires_at-now())):: double precision)
		end
		from public.urls
		where short_code=$1	
		`, shortUrl).Scan(&originalURL, &expired, &cacheTTL)
	if row != nil {
		if errors.Is(row, pgx.ErrNoRows) {
			return "", URL_NOT_FOUND
		}
		return "", fmt.Errorf("Database Error: %w", row)

	}
	//check expiration
	if expired {
		return "", URL_EXPIRED
	}
	ttl := time.Duration(cacheTTL * float64(time.Second))
	if ttl > 0 {
		row = database.Redis.Set(ctx, "url:"+shortUrl, originalURL, ttl).Err()

		if row != nil {
			fmt.Println("Redis cache error", row)
		}
	}
	//return orignal key
	return originalURL, nil

}
