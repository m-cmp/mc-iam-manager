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



