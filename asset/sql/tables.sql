-- 사용자 테이블
CREATE TABLE IF NOT EXISTS mcmp_users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 역할 테이블 (플랫폼 역할로 가정)
CREATE TABLE IF NOT EXISTS mcmp_platform_roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 사용자-플랫폼 역할 매핑 테이블
CREATE TABLE IF NOT EXISTS mcmp_user_platform_roles (
    user_id INTEGER REFERENCES mcmp_users(id) ON DELETE CASCADE,
    platform_role_id INTEGER REFERENCES mcmp_platform_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, platform_role_id)
);

-- 워크스페이스 역할 테이블 (신규 추가)
CREATE TABLE IF NOT EXISTS mcmp_workspace_roles (
    id SERIAL PRIMARY KEY,
    workspace_id VARCHAR(255) NOT NULL, -- 워크스페이스 식별자 (FK는 추후 정의)
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (workspace_id, name) -- 워크스페이스 내 역할 이름은 고유해야 함
);

-- 사용자-워크스페이스 역할 매핑 테이블 (신규 추가)
CREATE TABLE IF NOT EXISTS mcmp_user_workspace_roles (
    user_id INTEGER REFERENCES mcmp_users(id) ON DELETE CASCADE,
    workspace_role_id INTEGER REFERENCES mcmp_workspace_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, workspace_role_id)
);

-- 메뉴 테이블
CREATE TABLE IF NOT EXISTS mcmp_menu (
    id VARCHAR(255) PRIMARY KEY,
    parent_id VARCHAR(255) REFERENCES mcmp_menu(id) ON DELETE CASCADE,
    display_name VARCHAR(255) NOT NULL,
    res_type VARCHAR(50) NOT NULL,
    is_action BOOLEAN DEFAULT FALSE,
    priority INTEGER NOT NULL,
    menu_number INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 초기 메뉴 데이터 삽입 (DML 분리 예정)
-- INSERT INTO mcmp_menu (id, parent_id, display_name, res_type, is_action, priority, menu_number)
-- VALUES ('dashboard', NULL, 'Dashboard', 'menu', false, 1, 1)
-- ON CONFLICT (id) DO NOTHING;

-- 역할-권한 매핑 테이블 (role_menus -> mcmp_role_permissions 가정)
CREATE TABLE IF NOT EXISTS mcmp_role_permissions (
    role_type VARCHAR(50) NOT NULL, -- 'platform' or 'workspace'
    role_id INTEGER NOT NULL, -- mcmp_platform_roles.id 또는 mcmp_workspace_roles.id
    permission_id VARCHAR(255) NOT NULL, -- 권한 ID (별도 테이블 또는 정의 필요)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_type, role_id, permission_id)
    -- FK 제약 조건은 permission 테이블 정의 후 추가 필요
);

-- 권한 테이블 (신규 추가 예시)
CREATE TABLE IF NOT EXISTS mcmp_permissions (
    id VARCHAR(255) PRIMARY KEY,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
