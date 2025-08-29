package config

import (
	"database/sql"
	"fmt"
	"log"
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
	host := os.Getenv("MC_IAM_MANAGER_DATABASE_HOST")
	port := os.Getenv("MC_IAM_MANAGER_DATABASE_PORT")
	user := os.Getenv("MC_IAM_MANAGER_DATABASE_USER")
	password := os.Getenv("MC_IAM_MANAGER_DATABASE_PASSWORD")
	dbname := os.Getenv("MC_IAM_MANAGER_DATABASE_NAME")
	sslmode := os.Getenv("MC_IAM_MANAGER_DATABASE_SSLMODE")
	
	// 디버깅을 위한 로그 추가
	log.Printf("Database config - Host: %s, Port: %s, User: %s, DBName: %s, SSLMode: %s", 
		host, port, user, dbname, sslmode)
	
	return &DatabaseConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbname,
		SSLMode:  sslmode,
	}
}

// GetDSN 데이터베이스 연결 문자열 반환
func (c *DatabaseConfig) GetDSN() string {
	if dsn := os.Getenv("MC_IAM_MANAGER_DATABASE_URL"); dsn != "" {
		log.Printf("Using MC_IAM_MANAGER_DATABASE_URL: %s", dsn)
		return dsn
	}
	
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
	log.Printf("Generated DSN: %s", dsn)
	return dsn
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
