package model

import (
	"time"
)

// Company 회사 정보 모델 (싱글톤 — 플랫폼당 1개)
// table: mcmp_companies
type Company struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	Description    string    `gorm:"size:500" json:"description"`
	RealmName      string    `gorm:"size:255;uniqueIndex" json:"realm_name"`
	KcClientID     string    `gorm:"size:255" json:"kc_client_id"`
	KcClientSecret string    `gorm:"size:255" json:"-"` // 응답에서 제외
	Status         string    `gorm:"size:20;default:'active'" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName mcmp_companies 테이블 이름 반환
func (Company) TableName() string {
	return "mcmp_companies"
}

// ToResponse kc_client_secret 제외한 응답 DTO 반환
func (c *Company) ToResponse() *CompanyResponse {
	return &CompanyResponse{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		RealmName:   c.RealmName,
		KcClientID:  c.KcClientID,
		Status:      c.Status,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// CompanyResponse 회사 정보 응답 DTO (kc_client_secret 제외)
type CompanyResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RealmName   string    `json:"realm_name"`
	KcClientID  string    `json:"kc_client_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CompanyRequest 회사 생성 요청
type CompanyRequest struct {
	Name           string `json:"name" binding:"required"`
	RealmName      string `json:"realm_name" binding:"required"`
	KcClientID     string `json:"kc_client_id" binding:"required"`
	KcClientSecret string `json:"kc_client_secret" binding:"required"`
	Description    string `json:"description"`
}

// CompanyUpdateRequest 회사 수정 요청 (name, description만 변경 가능)
type CompanyUpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}
