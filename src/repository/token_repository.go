package repository

import (
	"log"
	"time"

	"github.com/m-cmp/mc-iam-manager/model"
	"gorm.io/gorm"
)

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{
		db: db,
	}
}

func (r *TokenRepository) SaveToken(userID string, token string, expiresIn int64) error {
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	tokenModel := &model.Token{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	query := r.db.Create(tokenModel)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("SaveToken SQL Query (ERROR): %s", sql)
		log.Printf("SaveToken SQL Args (ERROR): %v", args)
		return err
	}

	return nil
}

func (r *TokenRepository) GetTokenByUserID(userID string) (*model.Token, error) {
	var token model.Token
	query := r.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).First(&token)
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("GetTokenByUserID SQL Query (ERROR): %s", sql)
		log.Printf("GetTokenByUserID SQL Args (ERROR): %v", args)
		return nil, err
	}

	return &token, nil
}

func (r *TokenRepository) DeleteExpiredTokens() error {
	query := r.db.Where("expires_at < ?", time.Now()).Delete(&model.Token{})
	if err := query.Error; err != nil {
		// 에러 발생 시에만 쿼리 로깅
		sql := query.Statement.SQL.String()
		args := query.Statement.Vars
		log.Printf("DeleteExpiredTokens SQL Query (ERROR): %s", sql)
		log.Printf("DeleteExpiredTokens SQL Args (ERROR): %v", args)
		return err
	}

	return nil
}
