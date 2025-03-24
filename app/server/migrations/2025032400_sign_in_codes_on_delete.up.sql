ALTER TABLE sign_in_codes
DROP CONSTRAINT sign_in_codes_org_id_fkey,
ADD CONSTRAINT sign_in_codes_org_id_fkey
FOREIGN KEY (org_id)
REFERENCES orgs(id)
ON DELETE SET NULL;

