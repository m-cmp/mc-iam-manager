# test를 진행하기 위한 actor를 정의한다.

* actor는 model/User 를 참조한다. 정의된 json이름을 사용한다.
* profile은 사전정의된 platformRole을 참조한다.(PREDEFINED_PLATFORM_ROLE=admin,operator,viewer,billadmin,billviewer)

user profile1 : 관리자(admin)
- username : testadmin01
- email : testadmin01@test.com
- firstname : ta
- lastname : 01
- password : testadmin011111

user profile2 : 운영자(operator)
- username : testoperator01
- email : testoperator01@test.com
- firstname : to
- lastname : 01
- password : testoperator011111

user profile3 : 뷰어(viewer)
- username : testviewer01
- email : testviewer01@test.com
- firstname : tv
- lastname : 01
- password : testviewer011111

user profile4 : 재정관리자(billadmin)
- username : testbilladmin01
- email : testbilladmin01@test.com
- firstname : tba
- lastname : 01
- password : testbilladmin011111

user profile4 : 재정뷰어(billadmin)
- username : testbillviewer01
- email : testbillviewer01@test.com
- firstname : tbv
- lastname : 01
- password : testbillviewer011111

# test를 진행하기 위한 project를 정의한다.
project profile1
- projectname : testprj01
- projectdesc : testprj01 desc

project profile2
- projectname : testprj02
- projectdesc : testprj02 desc


# test를 진행하기 위한 workspace를 정의한다.
workspace profile1
- workspacename : testws01
- workspacedesc : testws01 desc

workspace profile2
- workspacename : testws02
- workspacedesc : testws02 desc


# test를 진행하기 위한 그룹용 사용자를 정의한다.

user profile6 : 그룹관리자(org-admin)
- username : orgadmin01
- email : orgadmin01@test.com
- firstname : oa
- lastname : 01
- password : orgadmin011111

user profile7 : 그룹멤버1(org-member)
- username : orgmember01
- email : orgmember01@test.com
- firstname : om
- lastname : 01
- password : orgmember011111

user profile8 : 그룹멤버2(org-member)
- username : orgmember02
- email : orgmember02@test.com
- firstname : om
- lastname : 02
- password : orgmember021111


# test를 진행하기 위한 그룹을 정의한다.

# seed 그룹 (groups.yaml에서 로딩)
group profile1 (seed - root)
- name : MZC
- group_code : 01
- description : M-CMP 최상위 그룹

group profile2 (seed - child)
- name : mc-iam-manager
- group_code : 0106
- parent : MZC(01)

group profile3 (seed - child)
- name : mc-infra-manager
- group_code : 0107
- parent : MZC(01)

# CRUD 테스트용 그룹
group profile4 (create)
- name : TestOrg-Dev
- description : 개발팀 테스트 그룹

group profile5 (create - child)
- name : TestOrg-Dev-Backend
- description : 백엔드팀 테스트 그룹
- parent : TestOrg-Dev


