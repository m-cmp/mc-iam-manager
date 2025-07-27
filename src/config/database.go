package config

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// DatabaseConfig 데이터베이스 설정
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewDatabaseConfig 데이터베이스 설정 생성
func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     os.Getenv("MC_IAM_MANAGER_DATABASE_HOST"),
		Port:     os.Getenv("MC_IAM_MANAGER_DATABASE_PORT"),
		User:     os.Getenv("MC_IAM_MANAGER_DATABASE_USER"),
		Password: os.Getenv("MC_IAM_MANAGER_DATABASE_PASSWORD"),
		DBName:   os.Getenv("MC_IAM_MANAGER_DATABASE_NAME"),
		SSLMode:  os.Getenv("MC_IAM_MANAGER_DATABASE_SSLMODE"),
	}
}

// GetDSN 데이터베이스 연결 문자열 반환
func (c *DatabaseConfig) GetDSN() string {
	if dsn := os.Getenv("MC_IAM_MANAGER_DATABASE_URL"); dsn != "" {
		return dsn
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

func InitDB() (*sql.DB, error) {
	host := os.Getenv("MC_IAM_MANAGER_DATABASE_HOST")
	port := os.Getenv("MC_IAM_MANAGER_DATABASE_PORT")
	user := os.Getenv("MC_IAM_MANAGER_DATABASE_USER")
	password := os.Getenv("MC_IAM_MANAGER_DATABASE_PASSWORD")
	dbname := os.Getenv("MC_IAM_MANAGER_DATABASE_NAME")
	sslmode := os.Getenv("MC_IAM_MANAGER_DATABASE_SSLMODE")

	if dsn := os.Getenv("MC_IAM_MANAGER_DATABASE_URL"); dsn != "" {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("데이터베이스 연결 실패: %v", err)
		}
		return db, nil
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("데이터베이스 연결 실패: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("데이터베이스 ping 실패: %v", err)
	}

	return db, nil
}
