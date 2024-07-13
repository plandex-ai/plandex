-- clean up any duplicates added mistakenly earlier
WITH ranked_duplicates AS (
  SELECT id,
         ROW_NUMBER() OVER (PARTITION BY org_id, user_id ORDER BY created_at) AS rn
  FROM orgs_users
)
DELETE FROM orgs_users
WHERE id IN (
  SELECT id FROM ranked_duplicates WHERE rn > 1
);
 
ALTER TABLE orgs_users ADD CONSTRAINT org_user_unique UNIQUE (org_id, user_id);
