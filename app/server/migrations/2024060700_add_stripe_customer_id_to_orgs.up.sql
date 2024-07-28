ALTER TABLE orgs ADD COLUMN stripe_customer_id VARCHAR(255); 
CREATE UNIQUE INDEX orgs_stripe_customer_id_idx ON orgs(stripe_customer_id);