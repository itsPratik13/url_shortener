package routes

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/url_shortener/database"
	"github.com/url_shortener/helpers"
)

type request struct {
	Url         string        `json:"url"  validate:"required,url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	Url             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	RateLimitRemain int           `json:"rate_limit"`
	RateLimitReset  time.Duration `json:"rate_limit_reset"`
}

const charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	ERR_SHORT_CODE_EXISTS            = errors.New("custom short code already exists in DB")
	ERR_SHORT_CODE_GENERATION_FAILED = errors.New("unable to generate a unique short code")
	//ERR_SHORT_CODE_COLLISION         = errors.New("generated already existing short code")
)

func GenerateShortCode(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
		if err != nil {
			return "", err
		}
		result[i] = charSet[n.Int64()]
	}
	return string(result), nil
}

func createShortURL(ctx context.Context, originalURL string, customShort string, expiry time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var expiresAt *time.Time
	if expiry > 0 {
		expiryTime := time.Now().Add(expiry) //take current time and add it by expiry given by user
		expiresAt = &expiryTime
	}
	//if user gives custom code
	if customShort != "" {
		err := database.DB.QueryRow(ctx, `
	insert into public.urls (short_code,original_url,expires_at)
	values($1,$2,$3)
	on conflict (short_code) do nothing
	returning short_code
	
	`, customShort, originalURL, expiresAt).Scan(&customShort)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ERR_SHORT_CODE_EXISTS

		}
		if err != nil {
			return "", fmt.Errorf("database error: %w", err)
		}
		return customShort, nil
	}
	//if user does not give custom code
	for i := 0; i < 5; i++ {
		// we check for 5 times if the code generated does not collide and is entered in db
		generatedShortCode, err := GenerateShortCode(7)
		if err != nil {
			return "", fmt.Errorf("short code generating error :%w ", err)
		}
		// try adding the generatedShortCode in DB
		err = database.DB.QueryRow(ctx,
			`insert into public.urls (short_code,original_url,expires_at)
		values($1,$2,$3)
		on conflict(short_code) do nothing
		returning short_code
		`, generatedShortCode, originalURL, expiresAt).Scan(&generatedShortCode)

		if err == nil {
			return generatedShortCode, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		return "", fmt.Errorf("database error: %w", err)

	}
	return "", ERR_SHORT_CODE_GENERATION_FAILED

}

var validate = validator.New()

func ShortenURL(ctx *fiber.Ctx) error {
	body := new(request) // already a type of *request  as new creates a pointer to request struct
	/*
		if declared as body:=request{} then pass &body as address in bodyparser
	*/
	//1. parse the body
	err := ctx.BodyParser(body)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	/* check if the input is an url*/

	if err := validate.Struct(body); err != nil {
		// validate.Struct(body) validates on tags
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid URL"})

	}
	// validate expiry
	if body.Expiry < 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "expiry cannot be negative"})
	}
	/* check if the input url is not localhost*/
	if !helpers.RemoveDomainError(body.Url) {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Invalid Domain"})

	}
	/* enforce https,ssl*/
	body.Url = helpers.EnforceHTTP(body.Url)

	shortCode, err := createShortURL(ctx.UserContext(), body.Url, body.CustomShort, body.Expiry)
	if errors.Is(err, ERR_SHORT_CODE_EXISTS) {
		return ctx.Status(fiber.StatusConflict).JSON(
			fiber.Map{"error": "Custom short URL already exists"},
		)
	}

	return ctx.Status(fiber.StatusInternalServerError).JSON(
		fiber.Map{"error": "Could not create short URL"},
	)
	shortURL := ctx.BaseURL() + "/" + shortCode

	return ctx.Status(fiber.StatusCreated).JSON(response{
		Url:             body.Url,
		CustomShort:     shortURL,
		Expiry:          body.Expiry,
		RateLimitRemain: 0,
		RateLimitReset:  0,
	})
}
