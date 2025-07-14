package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Claude pro and max use time-based cooldowns (4 hours, 8 hours, etc.) after reaching quota
// However, we still use a relatively short cooldown so that we find out fairly quickly if the quota has reset
const claudeSubscriptionCooldownDuration = 10 * time.Minute

type OrgUserConfig struct {
	PromptedClaudeMax                   bool      `json:"promptedClaudeMax"`
	UseClaudeSubscription               bool      `json:"useClaudeSubscription"`
	ClaudeSubscriptionCooldownStartedAt time.Time `json:"claudeSubscriptionCooldownStartedAt"`
}

func (p *OrgUserConfig) IsClaudeSubscriptionCooldownActive() bool {
	if p == nil || p.ClaudeSubscriptionCooldownStartedAt.IsZero() {
		return false // never started
	}
	return time.Since(p.ClaudeSubscriptionCooldownStartedAt) < claudeSubscriptionCooldownDuration
}

func (p *OrgUserConfig) Scan(src interface{}) error {
	if src == nil {
		*p = OrgUserConfig{}
		return nil
	}
	switch s := src.(type) {
	case []byte:
		if len(s) == 0 {
			*p = OrgUserConfig{}
			return nil
		}
		return json.Unmarshal(s, p)
	case string:
		if s == "" {
			*p = OrgUserConfig{}
			return nil
		}
		return json.Unmarshal([]byte(s), p)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (p *OrgUserConfig) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}
