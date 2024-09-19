CREATE TABLE IF NOT EXISTS sign_in_codes (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  pin_hash VARCHAR(64) NOT NULL,
  user_id UUID REFERENCES users(id),  
  org_id UUID REFERENCES orgs(id),
  auth_token_id UUID REFERENCES auth_tokens(id),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TRIGGER update_sign_in_codes_modtime BEFORE UPDATE ON sign_in_codes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE UNIQUE INDEX sign_in_codes_idx ON sign_in_codes(pin_hash, created_at DESC);