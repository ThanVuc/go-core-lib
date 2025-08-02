# go-core-lib

A comprehensive Go library providing core functionality for enterprise-grade applications, including structured logging, configuration management, and Redis caching capabilities.

## Features

- 🚀 **Structured Logging**: Zap-based logger with JSON output and environment-specific formatting
- ⚙️ **Configuration Management**: Viper-based configuration loader with environment support
- 🔧 **Redis Caching**: Type-safe Redis operations with generic support
- 📦 **Production Ready**: Optimized for containerized and cloud-native environments

## Installation

```bash
go get github.com/thanvuc/go-core-lib
```

## Quick Start

### 1. Logging

The logging package provides a structured logger built on top of Zap with request ID tracking and environment-specific formatting.

```go
package main

import (
    "github.com/thanvuc/go-core-lib/log"
    "go.uber.org/zap"
)

func main() {
    // Initialize logger
    logger := log.NewLogger(log.Config{
        Env:   "production", // or "dev" for development
        Level: "info",       // debug, info, warn, error
    })
    defer logger.Sync()

    // Log with request ID and additional fields
    requestID := "req-123456"
    logger.Info("User authenticated", requestID, 
        zap.String("user_id", "12345"),
        zap.String("email", "user@example.com"),
    )

    logger.Error("Database connection failed", requestID,
        zap.String("database", "postgres"),
        zap.Int("retry_count", 3),
    )
}
```

**Output in production (JSON format):**
```json
{
  "time": "2025-08-02T10:30:00Z",
  "level": "info",
  "caller": "main.go:15",
  "message": "User authenticated",
  "request_id": "req-123456",
  "env": "production",
  "user_id": "12345",
  "email": "user@example.com"
}
```

### 2. Configuration Management

Load configuration from YAML files with environment-specific support.

**config/dev.yaml:**
```yaml
database:
  host: localhost
  port: 5432
  name: myapp_dev
  
redis:
  addr: localhost:6379
  password: ""
  db: 0

log:
  level: debug
```

**config/production.yaml:**
```yaml
database:
  host: ${DB_HOST}
  port: ${DB_PORT}
  name: ${DB_NAME}
  
redis:
  addr: ${REDIS_ADDR}
  password: ${REDIS_PASSWORD}
  db: 0

log:
  level: info
```

```go
package main

import (
    "github.com/thanvuc/go-core-lib/config"
)

type AppConfig struct {
    Database struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
        Name string `mapstructure:"name"`
    } `mapstructure:"database"`
    
    Redis struct {
        Addr     string `mapstructure:"addr"`
        Password string `mapstructure:"password"`
        DB       int    `mapstructure:"db"`
    } `mapstructure:"redis"`
    
    Log struct {
        Level string `mapstructure:"level"`
    } `mapstructure:"log"`
}

func main() {
    var cfg AppConfig
    
    // Set GO_ENV environment variable to load specific config
    // GO_ENV=production loads production.yaml
    // GO_ENV=dev (or unset) loads dev.yaml
    err := config.LoadConfig(&cfg, "./config")
    if err != nil {
        panic(err)
    }
    
    // Use configuration
    fmt.Printf("Database: %s:%d/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
    fmt.Printf("Redis: %s\n", cfg.Redis.Addr)
}
```

### 3. Redis Caching

Type-safe Redis operations with generic support for any data type.

```go
package main

import (
    "time"
    "github.com/thanvuc/go-core-lib/cache"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Initialize Redis cache
    redisCache := cache.NewRedisCache(cache.Config{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    defer redisCache.Close()

    // Cast to access generic methods
    r := redisCache.(*cache.redisCache)

    // Set data with expiration
    user := User{ID: 1, Name: "John Doe", Email: "john@example.com"}
    err := cache.Set(r, "user:1", user, 1*time.Hour)
    if err != nil {
        panic(err)
    }

    // Get data
    retrievedUser, err := cache.Get[User](r, "user:1")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Retrieved user: %+v\n", retrievedUser)

    // Get and Set (cache-aside pattern)
    cachedUser, err := cache.GetAndSet(r, "user:2", User{
        ID: 2, Name: "Jane Doe", Email: "jane@example.com",
    }, 30*time.Minute)
    if err != nil {
        panic(err)
    }

    // Check if key exists
    exists, err := cache.Exists(r, "user:1")
    if err != nil {
        panic(err)
    }
    fmt.Printf("User exists: %v\n", exists)

    // Delete key
    err = cache.Delete(r, "user:1")
    if err != nil {
        panic(err)
    }
}
```

## Environment Configuration

The library supports environment-based configuration:

- **Development**: `GO_ENV=dev` or unset
  - Colored console output for logs
  - Debug-level logging with stack traces
  - Loads `dev.yaml` configuration

- **Production**: `GO_ENV=qa`
  - JSON structured logging
  - Optimized for log aggregation systems
  - Loads `qa.yaml` configuration

- **Production**: `GO_ENV=production`
  - JSON structured logging
  - Optimized for log aggregation systems
  - Loads `production.yaml` configuration

## Project Structure

```
go-core-lib/
├── cache/          # Redis caching functionality
│   ├── config.go   # Cache configuration
│   ├── interface.go # Cache interface definition
│   └── redis.go    # Redis implementation
├── config/         # Configuration management
│   └── config.go   # Viper-based config loader
├── log/            # Structured logging
│   ├── config.go   # Logger configuration
│   ├── interface.go # Logger interface
│   └── logger.go   # Zap-based implementation
└── eventbus/       # Event bus (planned)
    ├── consumer/
    ├── handler/
    └── producer/
```

## Dependencies

- [Zap](https://github.com/uber-go/zap): High-performance structured logging
- [Viper](https://github.com/spf13/viper): Configuration management
- [go-redis](https://github.com/redis/go-redis): Redis client
- [Lumberjack](https://github.com/natefinch/lumberjack): Log rotation

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License. 

