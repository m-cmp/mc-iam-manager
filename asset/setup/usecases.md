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

 
 ------------------------------
 1usecase.sh 를 실행 : 사용자 추가
   -> mcmp_users table에 추가한 유저가 있다
   -> keycloak console에 해당 user들이 추가되어있다. ( 아직 platform role은 매핑되어 있지 않다 )

 2usecase.sh 를 실행 : User에게 platform role 할당
   . realm role에 등록되지 않은 상태로 호출되는 경우가 있음. "error": "Failed to assign role: realm role operator not found"
   . 정상 등록 "message": "Successfully assigned role admin to user testadmin01"
   -> keycloak console에서 user > user선택 > role mapping tab에 platform role이 등록되어 있다.

