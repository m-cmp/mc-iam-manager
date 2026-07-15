# Role Permission Backup / Restore — 사용 방안

worktree: `mc-iam-manager/.worktrees/feat-iam-role-permission-backup`  
branch: `feat-iam-role-permission-backup`

## 목적

- **Backup**: DB에 실제로 걸려 있는 **역할 권한**(지금은 menus)을 `role-permission-backup` 문서로 보관
- **Restore**: 보관 문서로 역할 권한을 되돌림
- `permission.yaml` / initial 시드와 **별개** — 시드는 desired 템플릿, 백업은 actual 운영 상태

## API

| Method | Path | 설명 |
|---|---|---|
| `GET` | `/api/setup/backup-role-permissions` | 현재 역할 권한 백업 |
| `POST` | `/api/setup/restore-role-permissions` | 백업 문서 복구 |

인증: Platform Admin (`PlatformAdminMiddleware`)

### Backup

```bash
# 전체 플랫폼 역할, menus, YAML 응답
curl -sk -H "Authorization: Bearer $TOKEN" \
  'https://<host>:5000/api/setup/backup-role-permissions'

# 특정 역할 + 파일 저장 (asset/menu/backups/)
curl -sk -H "Authorization: Bearer $TOKEN" \
  'https://<host>:5000/api/setup/backup-role-permissions?roles=admin,operator&sections=menus&save=true' \
  -o role-permission-backup.yaml

# JSON
curl -sk -H "Authorization: Bearer $TOKEN" \
  'https://<host>:5000/api/setup/backup-role-permissions?format=json'
```

Query

| 파라미터 | 기본 | 설명 |
|---|---|---|
| `roles` | (전체 platform) | `admin,operator` |
| `sections` | `menus` | `menus` (`operations`/`csps`는 reserved, 빈 배열) |
| `format` | `yaml` | `yaml` \| `json` |
| `save` | false | `true` 시 `asset/menu/backups/role-permission-backup-*.yaml` 저장. 경로는 응답 헤더 `X-Role-Permission-Backup-Path` |

### Restore

```bash
# body에 YAML 전달 (additive: 없는 매핑만 추가)
curl -sk -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/yaml' \
  --data-binary @role-permission-backup.yaml \
  'https://<host>:5000/api/setup/restore-role-permissions?mode=additive&sections=menus'

# 서버 로컬 파일
curl -sk -X POST -H "Authorization: Bearer $TOKEN" \
  'https://<host>:5000/api/setup/restore-role-permissions?mode=replace-role&filePath=asset/menu/backups/role-permission-backup-20260715-120000.yaml'
```

| mode | 동작 |
|---|---|
| `additive` (기본) | 백업에 있는 (role, menu)만 없으면 추가. 기존 커스텀 유지 |
| `replace-role` | 백업에 등장한 role의 메뉴 매핑을 백업 집합으로 **교체**(기존 삭제 후 재생성) |

## 권장 운영 순서 (initial 재실행 전)

```
1) GET  .../backup-role-permissions?save=true
2) (선택) initial / permission 시드 재실행
3) 문제 시 POST .../restore-role-permissions?mode=additive|replace-role
4) 일상 변경은 메뉴-역할 개별 API 사용
```

## 백업 문서 예시

```yaml
kind: role-permission-backup
backupAt: "2026-07-15T13:00:00+09:00"
source: db
sections: [menus]
permissions:
  - role: admin
    menus: [operations, observability]
    operations: []
    csps: []
```

- 키는 **`role` 이름** (`role_masters.name`). numeric `role_id` 없음.
- 스키마 변경 없음 (`mcmp_role_menu_mappings`만 사용).

## 제한 (1단계)

- `operations` / `csps` section은 응답 자리만 유지 (시드/복구 미구현)
- replace-role은 해당 role의 **menus 전체**를 백업 기준으로 맞추므로 운영 커스텀이 삭제될 수 있음 → 먼저 backup 필수
