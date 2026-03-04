## 사용예시를 정의한다.
### usecase00은 사전작업이므로 무시한다.
### 관련 sampledata는 actors.md 에 정의한다.

//usecase00 : 플랫폼 관리자 추가. keycloak console에서 작업 후 .env 파일 갱신.(realm추가, client추가, role추가, user 추가)

usecase01 : 사용자 추가
 - user profile1~4까지 추가

usecase02 : platform Role 할당
 - user profile1 = admin
 - user profile2 = operator
 - user profile3 = viewer
 - user profile4 = billadmin
 - user profile5 = billviewer

usecase03 : workspace와 project mapping
 - workspace profile1~2 생성
 - project profile1~2 생성
 - workspace에 project 할당
   . testws01 - testprj01
   . testws01 - testprj02
 - workspace에서 project 할당 해제
   . testws01 - testprj02 할당 해제

usecase04 : platform role에 해당하는 작업 수행
 - user profile1~5가 menuTree 조회


usecase05 : workspace role 할당
 - user profile1 을 workspace profile1에 admin
 - user profile2 을 workspace profile1에 operator
 - user profile3 을 workspace profile1에 viewer
 - user profile4 을 workspace profile1에 billadmin
 - user profile5 을 workspace profile1에 billviewer

usecase06 : workspace role과 csp role 매핑
 - csp role 목록 조회
   . workspace role 목록과 csp role 목록 비교
 - csp role이 workspace role에 없으면 workspace role추가( csp role의 prefix는 mcmp_ )
   . workspace role : admin 과 csp role mcmp_admin
   . workspace role : operator 과 csp role mcmp_operator
   . workspace role : viewer 과 csp role mcmp_viewer
   . workspace role : billadmin 과 csp role mcmp_billadmin
   . workspace role : billviewer 과 csp role mcmp_billviewer

usecase07 : workspace role 관리
 - predefined된 workspace role은 삭제 불가.
 - 새로운 workspace role 추가
   . workspace role : observer -> csp role : mcmp_opserver
 - workspace role과 csp role 매핑
 - workspace role과 csp role 매핑해제

usecase08 : csp role 관리
 - 등록된 role에 permission 추가.( readonly to edit)
 - 등록된 role에 permission 추가.( vm to k8s)


usecase09 : api access 관리
 - api 등록
 - api resource 에 operationId(=action) 등록
 - workspace role에 따라 access 제어

usecase10 : 임시자격증명 발급 및 사용
 - 자신의 롤안에서 임시자격증명 발급하여 조회기능 수행
 - 임시자격증명으로 롤 밖 action 수행
 - 임시자격증명으로 생성,삭제기능 수행

usecase11 : 그룹 생성
 - seed 데이터 로딩
   . user profile1(admin)이 초기 그룹 데이터 로딩
   . POST /api/setup/initial-groups
   . GET /api/groups?tree=true 로 트리 확인
   . MZC(01) 하위 8개 프레임워크 그룹(0101~0108) 확인
 - 최상위 그룹 생성
   . group profile4(TestOrg-Dev) 생성
   . POST /api/groups {"name":"TestOrg-Dev", "description":"개발팀 테스트 그룹"}
   . group_code 자동생성 확인 (예: 02)
 - 하위 그룹 생성
   . group profile5(TestOrg-Dev-Backend) 생성
   . POST /api/groups {"name":"TestOrg-Dev-Backend", "parent_id": <TestOrg-Dev ID>}
   . group_code 자동생성 확인 (예: 0201)
 - 그룹 조회
   . GET /api/groups?tree=true 전체 트리 확인
   . GET /api/groups/id/:groupId 단건 조회
   . GET /api/groups/code/:code 코드로 조회
 - 그룹 수정
   . PUT /api/groups/id/:groupId {"name":"TestOrg-Dev-Updated"}
 - 그룹 삭제
   . 하위그룹 있을 때 삭제 시도 -> 실패(400) 확인
   . 하위그룹 제거 후 삭제 -> 성공(200) 확인

usecase12 : 그룹용 사용자 생성
 - user profile6~8 추가
   . user profile6(orgadmin01) 추가
   . user profile7(orgmember01) 추가
   . user profile8(orgmember02) 추가
 - platform role 할당
   . user profile6 = operator
   . user profile7 = viewer
   . user profile8 = viewer

usecase13 : 그룹에 사용자 추가
 - 사용자를 그룹에 할당
   . user profile1(admin)이 user profile6(orgadmin01)을 group profile1(MZC)에 할당
   . POST /api/users/id/<profile6 userId>/groups {"group_ids": [<MZC ID>]}
   . user profile7(orgmember01)을 group profile2(mc-iam-manager)에 할당
   . user profile8(orgmember02)을 group profile2(mc-iam-manager)에 할당
 - 다중 그룹 소속
   . user profile6(orgadmin01)을 group profile2(mc-iam-manager)에도 추가
   . user profile6이 MZC + mc-iam-manager 2개 그룹 소속
 - 그룹 소속 확인
   . GET /api/users/id/<profile6 userId>/groups -> MZC, mc-iam-manager 2개 확인
   . GET /api/groups/id/<mc-iam-manager ID>/users -> profile6, profile7, profile8 확인
 - 그룹에서 사용자 제거
   . DELETE /api/users/id/<profile6 userId>/groups/<MZC ID>
   . profile6에서 MZC 제거 후 mc-iam-manager만 소속 확인

usecase14 : 그룹 역할할당
 - 그룹에 platform role 할당
   . user profile1(admin)이 group profile2(mc-iam-manager)에 operator role 할당
   . POST /api/groups/id/:groupId/platform-roles {"role_id": <operator role ID>}
   . DB: mcmp_group_platform_roles에 저장
   . Keycloak: AddRealmRoleToGroup으로 그룹에 realm role 매핑
   . GET /api/groups/id/:groupId/platform-roles 로 할당 확인
 - 메뉴 자동 합산 확인
   . user profile7(orgmember01)은 개인 platform role = viewer (usecase12에서 할당)
   . user profile7은 mc-iam-manager 그룹 소속 (usecase13에서 할당)
   . user profile7 로그인 → JWT realm_access.roles에 viewer + operator 포함 확인
   . POST /api/users/menus-tree/list → viewer 메뉴 + operator 메뉴 합산 확인
 - 그룹에 workspace role 매핑
   . user profile1(admin)이 group profile2(mc-iam-manager)를 workspace profile1(testws01)에 매핑
   . POST /api/groups/id/:groupId/workspaces {"workspace_id": <testws01 ID>, "role_id": <viewer role ID>}
   . DB: mcmp_group_workspace_roles에 저장 (Keycloak 미사용)
   . mc-iam-manager 그룹 멤버(profile6,7,8)가 testws01에서 viewer 역할 자동 획득
 - 워크스페이스 매핑 조회
   . GET /api/groups/id/:groupId/workspaces → 매핑된 워크스페이스 + 역할 목록
 - 자동 접근 권한 확인
   . user profile7(orgmember01, mc-iam-manager 소속)이 testws01 접근 → viewer 역할로 접근 가능
   . 개인 UserWorkspaceRole 없이도 그룹 매핑으로 접근 가능 확인
 - 우선순위 확인: 개인 UserWorkspaceRole > 그룹 매핑 역할
   . user profile7에게 testws01에 operator 개인 할당 (usecase05 방식)
   . 그룹은 viewer, 개인은 operator → operator가 적용되는지 확인
 - 매핑 역할 변경
   . PUT /api/groups/id/:groupId/workspaces/:workspaceId {"role_id": <operator role ID>}
   . viewer → operator로 변경
 - 매핑 제거
   . DELETE /api/groups/id/:groupId/workspaces/:workspaceId
   . 제거 후 자동 접근 권한 해제 확인
 - platform role 해제
   . DELETE /api/groups/id/:groupId/platform-roles/:roleId
   . DB + Keycloak에서 제거
   . 그룹 멤버 재로그인 후 해당 role 메뉴 미표시 확인


