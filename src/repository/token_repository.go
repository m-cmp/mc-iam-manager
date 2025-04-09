package repository

import (
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

	return r.db.Create(tokenModel).Error
}

func (r *TokenRepository) GetTokenByUserID(userID string) (*model.Token, error) {
	var token model.Token
	err := r.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *TokenRepository) DeleteExpiredTokens() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.Token{}).Error
}
