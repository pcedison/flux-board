ALTER TABLE archived_tasks
  ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 2147483647;

-- W9/5-B enterprise seam:
-- if board rows later become workspace-scoped, the ordering guarantees in this
-- migration must widen only after every existing row is backfilled into a
-- default workspace. The future uniqueness target is expected to become
-- `(workspace_id, status, sort_order)`, not a route-only convention.

WITH ranked AS (
  SELECT
    id,
    status,
    ROW_NUMBER() OVER (
      PARTITION BY status
      ORDER BY sort_order ASC, last_updated DESC, id ASC
    ) - 1 AS normalized_sort_order
  FROM tasks
)
UPDATE tasks AS t
SET sort_order = ranked.normalized_sort_order
FROM ranked
WHERE ranked.id = t.id
  AND ranked.status = t.status
  AND t.sort_order IS DISTINCT FROM ranked.normalized_sort_order;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'tasks_status_allowed'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE tasks
      ADD CONSTRAINT tasks_status_allowed
      CHECK (status IN ('queued', 'active', 'done'));
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'tasks_priority_allowed'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE tasks
      ADD CONSTRAINT tasks_priority_allowed
      CHECK (priority IN ('critical', 'high', 'medium'));
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'tasks_due_format'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE tasks
      ADD CONSTRAINT tasks_due_format
      CHECK (due ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'tasks_sort_order_nonnegative'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE tasks
      ADD CONSTRAINT tasks_sort_order_nonnegative
      CHECK (sort_order >= 0);
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'tasks_status_sort_order_unique'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE tasks
      ADD CONSTRAINT tasks_status_sort_order_unique
      UNIQUE (status, sort_order)
      DEFERRABLE INITIALLY IMMEDIATE;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'archived_tasks_status_allowed'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE archived_tasks
      ADD CONSTRAINT archived_tasks_status_allowed
      CHECK (status IN ('queued', 'active', 'done'));
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'archived_tasks_priority_allowed'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE archived_tasks
      ADD CONSTRAINT archived_tasks_priority_allowed
      CHECK (priority IN ('critical', 'high', 'medium'));
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'archived_tasks_due_format'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE archived_tasks
      ADD CONSTRAINT archived_tasks_due_format
      CHECK (due ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$');
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class r ON r.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = r.relnamespace
    WHERE c.conname = 'archived_tasks_sort_order_nonnegative'
      AND n.nspname = current_schema()
  ) THEN
    ALTER TABLE archived_tasks
      ADD CONSTRAINT archived_tasks_sort_order_nonnegative
      CHECK (sort_order >= 0);
  END IF;
END $$;
