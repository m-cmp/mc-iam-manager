# Role Permission Backup / Restore — Test Cases

## 단위 테스트 (자동)

위치: `src/service/role_permission_backup_test.go`

| TC ID | 시나리오 | 기대 |
|---|---|---|
| TC-RPB-U01 | admin에 menus 매핑 후 Backup | kind=`role-permission-backup`, role=admin, menus 정렬 목록 |
| TC-RPB-U02 | Backup → 매핑 삭제 → Restore additive(+menu) | MenusAdded=3, DB에 3개 매핑 |
| TC-RPB-U03 | 3개 매핑 → Restore replace-role(2개) | Removed=3, Added=2, DB는 백업 2개만 |
| TC-RPB-U04 | YAML Parse | viewer / operations 파싱 성공 |

실행:

```bash
cd mc-iam-manager/.worktrees/feat-iam-role-permission-backup/src
go test ./service/ -run 'TestBackupAndRestore|TestRestoreRolePermissions_Replace|TestParseRolePermission' -count=1
```

## API TC (수동 / 통합)

전제: IAM 기동, platform admin 토큰, 플랫폼 역할·메뉴 시드됨.

| TC ID | 절차 | 기대 |
|---|---|---|
| TC-RPB-A01 | `GET /backup-role-permissions` | 200, YAML, `kind: role-permission-backup` |
| TC-RPB-A02 | `GET ...?roles=admin&format=json` | 200 JSON, permissions[0].role=admin |
| TC-RPB-A03 | `GET ...?save=true` | 200 + 헤더 `X-Role-Permission-Backup-Path`, 파일 존재 |
| TC-RPB-A04 | body YAML로 `POST /restore-role-permissions?mode=additive` | 200, menusAdded ≥ 0 |
| TC-RPB-A05 | `mode=replace-role` + 축소된 menus 백업 | 해당 role 메뉴가 백업과 일치 |
| TC-RPB-A06 | `filePath=` 로컬 백업 | 200, body 없이 복구 |
| TC-RPB-A07 | body/filePath 없음 | 400 |
| TC-RPB-A08 | 존재하지 않는 role 이름 | 500 (role not found) |
| TC-RPB-A09 | non-admin 토큰 | 401/403 |

## 회귀 체크리스트

- [ ] 기존 `GET /api/setup/initial-role-menu-permission` (CSV) 동작 유지
- [ ] Backup이 다른 역할 매핑을 변경하지 않음
- [ ] Additive 재호출 시 중복 행 증가 없음 (이미 있으면 skip)
- [ ] Replace 후 백업에 없는 메뉴 매핑 제거 확인

## 결과 기록

| TC ID | 결과 | 일자 | 비고 |
|---|---|---|---|
| TC-RPB-U01~U04 | PASS | 2026-07-15 | go test ./service/ -run TestBackup… |
| TC-RPB-A01~A09 | (환경 검증 후 기입) | | |
