package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/stretchr/testify/assert"
)

// fakePlatformAdminCheck HasPlatformAdminUser 만 구현한 KeycloakService (다른 메서드는 호출하지 않는다)
type fakePlatformAdminCheck struct {
	service.KeycloakService
	has bool
	err error
}

func (f fakePlatformAdminCheck) HasPlatformAdminUser(context.Context, string) (bool, error) {
	return f.has, f.err
}

// IAM-BUG-014: 설치가 끝난 뒤(platformAdmin 사용자가 있으면) initial-admin 은 403
func TestInitialAdminAlreadyDone(t *testing.T) {
	blocked, _ := initialAdminAlreadyDone(context.Background(), fakePlatformAdminCheck{has: false}, "t")
	assert.False(t, blocked, "클린 설치는 진행")

	blocked, resp := initialAdminAlreadyDone(context.Background(), fakePlatformAdminCheck{has: true}, "t")
	assert.True(t, blocked)
	assert.Equal(t, http.StatusForbidden, resp.status)
	assert.Contains(t, resp.body.Message, "already initialized")

	blocked, resp = initialAdminAlreadyDone(context.Background(), fakePlatformAdminCheck{err: errors.New("kc down")}, "t")
	assert.True(t, blocked, "확인 실패 시 진행하지 않는다")
	assert.Equal(t, http.StatusInternalServerError, resp.status)
}
