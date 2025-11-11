package database

import (
	"database/sql"
	"go-chat/internal/config"
	"go-chat/internal/logging"

	"github.com/go-redis/redis/v8"
)

type Database struct {
	RedisDB *redis.Client
	MySqlDB *sql.DB
}

func ConnectDatabase(logger *logging.Logger, cfg *config.Config) (*Database, error) {
	logger.Info("Connecting to Redis")
	redisDb, err := connectRedis(cfg.RedisURL)
	if err != nil {
		logger.Error("%s", err.Error())
		return nil, err
	}
	logger.Info("Connected to Redis successfully")

	logger.Info("Connecting to MySQL")
	mysqlDb, err := NewMySQLClient(cfg.MySqlDsn)
	if err != nil {
		logger.Error("failed to connect to MySQL: %v", err)
		return nil, err
	}
	logger.Info("Connected to MySQL successfully")

	return &Database{
		RedisDB: redisDb,
		MySqlDB: mysqlDb,
	}, nil
}
