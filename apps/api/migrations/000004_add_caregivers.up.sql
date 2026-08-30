-- Migration: 000004_add_caregivers.up.sql
-- Description: 新增照護人員主檔，單位透過 site_id 關聯既有據點；site_name_raw 保留
-- 匯入時比對不到據點的原始名稱，待人工於「待維護」畫面補建關聯。類型為固定二選一
-- （個管／專護），與姓名同為必填。

CREATE TABLE caregivers (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id       UUID REFERENCES sites(id),
    site_name_raw TEXT,
    name          TEXT NOT NULL,
    type          TEXT NOT NULL CHECK (type IN ('case_manager', 'specialist')),
    contact       TEXT,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_caregivers_site_id ON caregivers(site_id);
