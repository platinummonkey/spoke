-- Rollback Migration 007: Analytics Events

-- Drop partitions first
DROP TABLE IF EXISTS compilation_events_2026_03;
DROP TABLE IF EXISTS compilation_events_2026_02;
DROP TABLE IF EXISTS compilation_events_2026_01;

DROP TABLE IF EXISTS module_view_events_2026_03;
DROP TABLE IF EXISTS module_view_events_2026_02;
DROP TABLE IF EXISTS module_view_events_2026_01;

DROP TABLE IF EXISTS download_events_2026_03;
DROP TABLE IF EXISTS download_events_2026_02;
DROP TABLE IF EXISTS download_events_2026_01;

-- Drop parent tables (automatically drops all indexes)
DROP TABLE IF EXISTS compilation_events;
DROP TABLE IF EXISTS module_view_events;
DROP TABLE IF EXISTS download_events;
