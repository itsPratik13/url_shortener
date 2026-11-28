package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	DB    *pgxpool.Pool
	Redis *redis.Client
)

func ConnectPostgres(dbUrl string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("failed to parse postgres config : %w", err)
	}
	DB, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to connect with postgres: %w", err)
	}
	if err := DB.Ping(ctx); err != nil {
		DB.Close()
		DB = nil
		return fmt.Errorf("psotgres ping failed, %w", err)
	}
	log.Println("PostgreSql connected successfully")
	return nil
}

func ConnectRedis(addr, password string, db int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	Redis = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	err := Redis.Ping(ctx).Err()
	if err != nil {
		Redis.Close()
		Redis = nil
		return fmt.Errorf("Memurai connection failed: %w", err)
	}

	log.Println("Memurai connected successfully")
	return nil
}

func CloseConnections(){
	if DB!=nil{
		DB.Close()
	}
	if Redis != nil {
		if err := Redis.Close(); err != nil {
			log.Printf("Error closing Memurai: %v", err)
		}
	}
}
