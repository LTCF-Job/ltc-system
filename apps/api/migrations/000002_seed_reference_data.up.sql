-- Migration: 000002_seed_reference_data.up.sql
-- Description: 載入正式參考資料（全台灣 22 縣市）與預設 Supabase Auth 管理員帳號

INSERT INTO regions (name, description, status, sort_order) VALUES
('新竹縣', '新竹縣營運區域', 'active', 1),
('新竹市', '新竹市營運區域', 'active', 2),
('苗栗縣', '苗栗縣營運區域', 'active', 3),
('臺北市', '臺北市營運區域', 'active', 4),
('新北市', '新北市營運區域', 'active', 5),
('基隆市', '基隆市營運區域', 'active', 6),
('桃園市', '桃園市營運區域', 'active', 7),
('臺中市', '臺中市營運區域', 'active', 8),
('彰化縣', '彰化縣營運區域', 'active', 9),
('南投縣', '南投縣營運區域', 'active', 10),
('雲林縣', '雲林縣營運區域', 'active', 11),
('嘉義市', '嘉義市營運區域', 'active', 12),
('嘉義縣', '嘉義縣營運區域', 'active', 13),
('臺南市', '臺南市營運區域', 'active', 14),
('高雄市', '高雄市營運區域', 'active', 15),
('屏東縣', '屏東縣營運區域', 'active', 16),
('宜蘭縣', '宜蘭縣營運區域', 'active', 17),
('花蓮縣', '花蓮縣營運區域', 'active', 18),
('臺東縣', '臺東縣營運區域', 'active', 19),
('澎湖縣', '澎湖縣營運區域', 'active', 20),
('金門縣', '金門縣營運區域', 'active', 21),
('連江縣', '連江縣營運區域', 'active', 22)
ON CONFLICT (name) DO UPDATE SET
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  sort_order = EXCLUDED.sort_order;

INSERT INTO auth.users (
    id,
    instance_id,
    aud,
    role,
    email,
    encrypted_password,
    email_confirmed_at,
    raw_app_meta_data,
    raw_user_meta_data,
    created_at,
    updated_at,
    confirmation_token,
    recovery_token,
    email_change_token_new,
    email_change
)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000000',
    'authenticated',
    'authenticated',
    'ltcf-admin@ltc.example.com',
    '$2a$10$IKiM.bIaxDYbD/.cl.b8MODthsuQ0WhLDcvox90gC3H3TDHaHFVYe',
    now(),
    '{"provider":"email","providers":["email"],"role":"admin"}'::jsonb,
    '{"display_name":"系統管理員"}'::jsonb,
    now(),
    now(),
    '', '', '', ''
)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    encrypted_password = EXCLUDED.encrypted_password,
    email_confirmed_at = EXCLUDED.email_confirmed_at,
    raw_app_meta_data = EXCLUDED.raw_app_meta_data,
    raw_user_meta_data = EXCLUDED.raw_user_meta_data,
    updated_at = now();
