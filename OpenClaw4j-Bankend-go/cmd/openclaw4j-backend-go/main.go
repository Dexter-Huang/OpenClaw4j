package main

import (
	"context"
	"fmt"
	"log"

	hertz "github.com/cloudwego/hertz/pkg/app/server"
	hertzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/seaskyland/openclaw4j-backend-go/internal/bootstrap"
	appconfig "github.com/seaskyland/openclaw4j-backend-go/internal/config"
	"github.com/seaskyland/openclaw4j-backend-go/internal/dao/redisdao"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDatabase,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	app, err := bootstrap.New(bootstrap.Options{
		DB:            pool,
		TokenSessions: redisdao.NewTokenSessionDAO(redisClient),
		HertzOptions:  []hertzconfig.Option{hertz.WithHostPorts(cfg.HTTPAddr)},
	})
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}

	app.HTTP.Spin()
}
