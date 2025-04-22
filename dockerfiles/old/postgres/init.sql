-- 사용자 테이블
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 역할 테이블
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 사용자-역할 매핑 테이블
CREATE TABLE IF NOT EXISTS user_roles (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- 메뉴 테이블
CREATE TABLE IF NOT EXISTS menus (
    id VARCHAR(255) PRIMARY KEY,
    parent_id VARCHAR(255) REFERENCES menus(id) ON DELETE CASCADE,
    display_name VARCHAR(255) NOT NULL,
    res_type VARCHAR(50) NOT NULL,
    is_action BOOLEAN DEFAULT FALSE,
    priority INTEGER NOT NULL,
    menu_number INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 초기 메뉴 데이터 삽입
INSERT INTO menus (id, parent_id, display_name, res_type, is_action, priority, menu_number)
VALUES ('dashboard', NULL, 'Dashboard', 'menu', false, 1, 1)
ON CONFLICT (id) DO NOTHING;

-- 역할-메뉴 매핑 테이블
CREATE TABLE IF NOT EXISTS role_menus (
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    menu_id VARCHAR(255) REFERENCES menus(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, menu_id)
); 