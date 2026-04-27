# Project ↔ Namespace 동기화 관리 API 설계 요약

브랜치: `feat-iam-ws-044`

## 신규 엔드포인트

### GET /api/setup/projects/sync-diff
- 인증: PlatformAdminMiddleware
- mc-infra-manager namespace vs 로컬 project 불일치 목록 반환 (read-only)
- 응답: `{ missingProjects: [...], unassignedProjects: [...] }`

### POST /api/setup/projects/sync
- 인증: PlatformAdminMiddleware
- 요청: `{ workspaceId: "N", nsIds: ["..."] }`
- 동작: 선택한 nsId를 로컬에 생성하거나 지정 workspace에 할당 (best-effort)
- 응답: `{ created, assigned, skipped, failed }`

## 변경 파일
- `src/model/request.go` — DTO 9개 추가
- `src/service/project_service.go` — `GetProjectSyncDiff`, `ApplyProjectSyncToWorkspace` 추가
- `src/handler/project_handler.go` — `GetProjectSyncDiff`, `ApplyProjectSync` 핸들러 추가
- `src/main.go` — setup 그룹에 두 라우트 등록
- `conf/mc-iam-manager/service-actions.yaml`, `asset/mcmpapi/service-actions.yaml` — 액션 2개 추가
- `src/docs/` — swagger 재생성

## 주의
- project 생성 시 `service.Create` 사용 금지 (mc-infra-manager PostNs 이중 호출). `projectRepo.CreateProject` 직접 사용.
- 기존 `POST /api/setup/sync-projects` 동작 변경 없음.

## 상세 문서
- 분석: `mc-iam-manager-spec/mc-iam-manager/docs/features/FR-WS-044-PROJECT-NAMESPACE-SYNC-ANALYSIS.md`
- 구현: `mc-iam-manager-spec/mc-iam-manager/docs/features/FR-WS-044-PROJECT-NAMESPACE-SYNC-IMPLEMENTATION.md`
- 테스트: `mc-iam-manager-spec/mc-iam-manager/docs/features/FR-WS-044-PROJECT-NAMESPACE-SYNC-TEST-RESULTS.md`
- 다이어그램: `mc-iam-manager-spec/mc-iam-manager/docs/diagrams/admin-setup/GET-projects-sync-diff.md`
- 다이어그램: `mc-iam-manager-spec/mc-iam-manager/docs/diagrams/admin-setup/POST-projects-sync.md`
