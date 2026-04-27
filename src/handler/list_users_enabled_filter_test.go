package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-M2-001-01: enabled=false 바인딩
func TestUserListRequest_EnabledFalse(t *testing.T) {
	raw := `{"enabled": false}`
	var req UserListRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &req))
	require.NotNil(t, req.Enabled, "enabled 필드가 *bool 포인터로 파싱되어야 한다")
	assert.False(t, *req.Enabled)
}

// TC-M2-001-02: enabled=true 바인딩
func TestUserListRequest_EnabledTrue(t *testing.T) {
	raw := `{"enabled": true}`
	var req UserListRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &req))
	require.NotNil(t, req.Enabled)
	assert.True(t, *req.Enabled)
}

// TC-M2-001-03: enabled 필드 미포함 → nil (전체 조회, 하위 호환)
func TestUserListRequest_EnabledOmitted(t *testing.T) {
	raw := `{}`
	var req UserListRequest
	require.NoError(t, json.Unmarshal([]byte(raw), &req))
	assert.Nil(t, req.Enabled, "enabled 미포함 시 nil이어야 한다 (전체 사용자 조회)")
}

// TC-M2-001-04: 빈 바디(nil) → nil
func TestUserListRequest_EmptyBody(t *testing.T) {
	var req UserListRequest
	assert.Nil(t, req.Enabled)
}
