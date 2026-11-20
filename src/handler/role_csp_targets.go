package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/constants"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
	"github.com/m-cmp/mc-iam-manager/util"
)

// 여정 1: Master Role 저장 시 대상 CSP Role 처리 (IAM-BUG-043, IAM-TECH-045 M1)
// Master Role 을 먼저 저장하고, 대상 CSP Role 은 항목별로 처리한다. 실패한 항목은 매핑만 빠지고 응답에 결과·경고로 남는다.

// cspRoleResult cspRoleResults[] 한 건
type cspRoleResult struct {
	CspRoleName string               `json:"cspRoleName"`
	CspType     string               `json:"cspType"`
	AuthMethod  constants.AuthMethod `json:"authMethod"`
	Result      string               `json:"result"`
	CspRoleID   uint                 `json:"cspRoleId,omitempty"`
	ErrorCode   string               `json:"errorCode,omitempty"` // invalid_request | protected | csp_error
	Error       string               `json:"error,omitempty"`
}

// roleSaveResponse Master Role 저장 응답. 역할 필드는 그대로 두고 cspRoleResults·warnings 를 덧붙인다.
type roleSaveResponse struct {
	*model.RoleMaster
	CspRoleResults []cspRoleResult `json:"cspRoleResults"`
	Warnings       []string        `json:"warnings"`
}

func newRoleSaveResponse(role *model.RoleMaster) *roleSaveResponse {
	return &roleSaveResponse{RoleMaster: role, CspRoleResults: []cspRoleResult{}, Warnings: []string{}}
}

func cspRoleKey(name, cspType string, authMethod constants.AuthMethod) string {
	if authMethod == "" {
		authMethod = constants.AuthMethodOIDC
	}
	return fmt.Sprintf("%s|%s|%s", cspType, name, authMethod)
}

// precheckCspRoleRequests Master Role 을 저장하기 전에 요청 값만 검사한다. cspRoleName 이 비면 400 (자동 명명 없음).
// 통과하면 이름을 mciam- 기준으로 정규화해 둔다.
func precheckCspRoleRequests(reqs []model.CreateCspRoleRequest) error {
	seen := map[string]bool{}
	for i := range reqs {
		name, err := service.NormalizeTargetCspRoleName(reqs[i].CspRoleName)
		if err != nil {
			return fmt.Errorf("cspRoles[%d]: %v", i, err)
		}
		if reqs[i].CspType == "" {
			return fmt.Errorf("cspRoles[%d]: cspType is required", i)
		}
		reqs[i].CspRoleName = name
		key := cspRoleKey(name, reqs[i].CspType, reqs[i].AuthMethod)
		if seen[key] {
			return fmt.Errorf("cspRoles[%d]: duplicate CSP role %s (%s)", i, name, reqs[i].CspType)
		}
		seen[key] = true
	}
	return nil
}

func failedCspRoleResult(r cspRoleResult, err error) cspRoleResult {
	r.Result = service.TargetResultFailed
	r.Error = err.Error()
	switch {
	case service.IsCspRoleProtectedError(err):
		r.ErrorCode = "protected"
	case service.IsCspRoleRequestError(err):
		r.ErrorCode = "invalid_request"
	default:
		r.ErrorCode = "csp_error"
	}
	return r
}

// applyCspRoleTargets 요청한 대상 CSP Role 을 Master Role 에 매핑한다.
//   - existing: 이미 걸린 매핑. 요청에도 있으면 unchanged, removeMissing 이면 요청에 없는 것은 해제(unmapped)
//   - 새 항목: ResolveTargetCspRole(DB → AWS 등록 또는 생성 → 매핑 검증) 후 매핑. 실패하면 failed, 매핑 없음
func (h *RoleHandler) applyCspRoleTargets(roleID uint, reqs []model.CreateCspRoleRequest, existing []service.RoleCspMapping, removeMissing bool, description string, resp *roleSaveResponse) {
	requested := map[string]bool{}
	for _, r := range reqs {
		requested[cspRoleKey(r.CspRoleName, r.CspType, r.AuthMethod)] = true
	}
	current := map[string]service.RoleCspMapping{}
	for _, m := range existing {
		current[cspRoleKey(m.CspRoleName, m.CspType, m.AuthMethod)] = m
	}

	if removeMissing {
		for key, m := range current {
			if requested[key] {
				continue
			}
			r := cspRoleResult{CspRoleName: m.CspRoleName, CspType: m.CspType, AuthMethod: m.AuthMethod, CspRoleID: m.CspRoleID, Result: service.TargetResultUnmapped}
			if err := h.roleService.DeleteRoleCspRoleMapping(roleID, m.CspRoleID, m.AuthMethod); err != nil {
				r = failedCspRoleResult(r, fmt.Errorf("failed to remove mapping: %w", err))
				resp.Warnings = append(resp.Warnings, fmt.Sprintf("%s: %s", m.CspRoleName, r.Error))
			}
			resp.CspRoleResults = append(resp.CspRoleResults, r)
		}
	}

	for i := range reqs {
		req := reqs[i]
		authMethod := req.AuthMethod
		if authMethod == "" {
			authMethod = constants.AuthMethodOIDC
		}
		r := cspRoleResult{CspRoleName: req.CspRoleName, CspType: req.CspType, AuthMethod: authMethod}

		if m, ok := current[cspRoleKey(req.CspRoleName, req.CspType, authMethod)]; ok {
			r.CspRoleID, r.Result = m.CspRoleID, service.TargetResultUnchanged
			resp.CspRoleResults = append(resp.CspRoleResults, r)
			continue
		}

		role, result, err := h.cspRoleService.ResolveTargetCspRole(&req)
		if role != nil {
			r.CspRoleID = role.ID
		}
		if err == nil {
			err = h.roleService.AddCspRolesMapping(&model.CreateRoleMasterCspRoleMappingRequest{
				RoleID:      util.UintToString(roleID),
				CspRoleID:   util.UintToString(role.ID),
				AuthMethod:  authMethod,
				Description: description,
			})
			if err != nil {
				err = fmt.Errorf("failed to map CSP role: %w", err)
			}
		}
		if err != nil {
			r = failedCspRoleResult(r, err)
			log.Printf("[WARN] role %d: CSP role %s (%s) not mapped: %v", roleID, req.CspRoleName, req.CspType, err)
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("%s: not mapped: %s", req.CspRoleName, r.Error))
		} else {
			r.Result = result
		}
		resp.CspRoleResults = append(resp.CspRoleResults, r)
	}
}

// checkCspRolesRequest POST /roles/csp/check 요청
type checkCspRolesRequest struct {
	CspRoles []model.CreateCspRoleRequest `json:"cspRoles"`
}

// CheckCspRoles godoc
// @Summary Check target CSP roles
// @Description 대상 CSP Role 이 IAM(DB)·CSP 에 있는지만 확인한다. DB·CSP 에 쓰지 않는다.
// @Tags roles
// @Accept json
// @Produce json
// @Param request body checkCspRolesRequest true "cspRoles (cspRoleName, cspType, authMethod, cspIdpConfigId)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/roles/csp/check [post]
// @Id checkCspRoles
func (h *RoleHandler) CheckCspRoles(c echo.Context) error {
	var req checkCspRolesRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	if len(req.CspRoles) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cspRoles is required"})
	}
	results := make([]service.CspRoleCheckResult, 0, len(req.CspRoles))
	for _, r := range req.CspRoles {
		results = append(results, h.cspRoleService.CheckTargetCspRole(r))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"results": results})
}
