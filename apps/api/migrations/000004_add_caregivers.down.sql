-- Migration: 000004_add_caregivers.down.sql
-- Description: 移除照護人員主檔。

DROP TABLE IF EXISTS caregivers;
