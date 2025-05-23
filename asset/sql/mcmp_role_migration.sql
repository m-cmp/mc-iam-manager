-- 기존 테이블 삭제
DROP TABLE IF EXISTS mcmp_user_platform_roles CASCADE;
DROP TABLE IF EXISTS mcmp_user_workspace_roles CASCADE;
DROP TABLE IF EXISTS mcmp_platform_roles CASCADE;
DROP TABLE IF EXISTS mcmp_workspace_roles CASCADE;

-- 새로운 역할 마스터 테이블 생성
CREATE TABLE mcmp_role_master (
    id SERIAL PRIMARY KEY,
    parent_id INT,
    name VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(1000),
    predefined BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (parent_id) REFERENCES mcmp_role_master(id) ON DELETE SET NULL
);

-- 새로운 역할 서브 테이블 생성
CREATE TABLE mcmp_role_sub (
    id SERIAL PRIMARY KEY,
    role_id INT NOT NULL,
    role_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (role_id) REFERENCES mcmp_role_master(id) ON DELETE CASCADE,
    UNIQUE(role_id, role_type)
);

-- 새로운 사용자-역할 매핑 테이블 생성
CREATE TABLE mcmp_user_platform_roles (
    user_id INT NOT NULL,
    role_id INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES mcmp_users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES mcmp_role_master(id) ON DELETE CASCADE
);

-- 새로운 사용자-워크스페이스-역할 매핑 테이블 생성
CREATE TABLE mcmp_user_workspace_roles (
    user_id INT NOT NULL,
    workspace_id INT NOT NULL,
    role_id INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, workspace_id, role_id),
    FOREIGN KEY (user_id) REFERENCES mcmp_users(id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id) REFERENCES mcmp_workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES mcmp_role_master(id) ON DELETE CASCADE
);

-- 기존 데이터 마이그레이션
-- 1. 플랫폼 역할 마이그레이션
INSERT INTO mcmp_role_master (name, description, created_at, updated_at)
SELECT name, description, created_at, updated_at
FROM mcmp_platform_roles;

-- 2. 워크스페이스 역할 마이그레이션
INSERT INTO mcmp_role_master (name, description, created_at, updated_at)
SELECT name, description, created_at, updated_at
FROM mcmp_workspace_roles;

-- 3. 플랫폼 역할 서브 타입 생성
INSERT INTO mcmp_role_sub (role_id, role_type, created_at)
SELECT id, 'platform', created_at
FROM mcmp_role_master
WHERE name IN (SELECT name FROM mcmp_platform_roles);

-- 4. 워크스페이스 역할 서브 타입 생성
INSERT INTO mcmp_role_sub (role_id, role_type, created_at)
SELECT id, 'workspace', created_at
FROM mcmp_role_master
WHERE name IN (SELECT name FROM mcmp_workspace_roles);

-- 5. 사용자-플랫폼 역할 매핑 마이그레이션
INSERT INTO mcmp_user_platform_roles (user_id, role_id, created_at)
SELECT upr.user_id, rm.id, upr.created_at
FROM mcmp_user_platform_roles upr
JOIN mcmp_platform_roles pr ON upr.platform_role_id = pr.id
JOIN mcmp_role_master rm ON pr.name = rm.name;

-- 6. 사용자-워크스페이스-역할 매핑 마이그레이션
INSERT INTO mcmp_user_workspace_roles (user_id, workspace_id, role_id, created_at)
SELECT uwr.user_id, uwr.workspace_id, rm.id, uwr.created_at
FROM mcmp_user_workspace_roles uwr
JOIN mcmp_workspace_roles wr ON uwr.workspace_role_id = wr.id
JOIN mcmp_role_master rm ON wr.name = rm.name; 