package routes

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/url_shortener/helpers"
)

type request struct {
	Url         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	Url             string        `json:"url" validate:"required,url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	RateLimitRemain int           `json:"rate_limit"`
	RateLimitReset  time.Duration `json:"rate_limit_reset"`
}
	var validate =validator.New()

func ShortenURL(ctx *fiber.Ctx) error{
	body:=new(request)  // already a type of *request  as new creates a pointer to request struct
	/*
	if declared as body:=request{} then pass &body as address in bodyparser
	*/
	err:=ctx.BodyParser(body)
	if err!=nil{
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"cannot parse JSON"})
	}

	/* implement rate limiting*/
	/* check if the input is an url*/

	if err:=validate.Struct(body); err!=nil{ 
	// validate.Struct(body) validates on tags
	return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error":"Invalid URL"})

	}
	/* check if the input url is not localhost*/
	if !helpers.RemoveDomainError(body.Url){
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error":"Invalid Domain"})

	}
	/* enforce https,ssl*/
	body.Url=helpers.EnforceHTTP(body.Url)
}