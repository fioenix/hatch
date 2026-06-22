// Package slack bridges the Hatch chat bus to a real Slack channel. Each agent
// maps to its own Slack app/bot, so it posts and is @mentioned as a real Slack
// principal; a single "hub" app runs Socket Mode (inbound) and is the fallback
// voice for escalations and any agent without its own token. It is a
// presentation/ingress adapter — it never orchestrates work; the boss in Slack
// is just a peer who happens to sit outside the terminal.
package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Config holds the bridge's routing and the boss's own Slack credentials.
// Tokens are loaded from the environment first, then a gitignored
// .hatch/slack/config.json. They are never committed.
type Config struct {
	AppToken  string            `json:"app_token"`  // xapp- (hub app, Socket Mode / inbound)
	HubToken  string            `json:"hub_token"`  // xoxb- (hub app: escalations + impersonation fallback)
	ChannelID string            `json:"channel_id"` // the #squad channel (C...)
	Boss      string            `json:"boss"`       // member id of the human (Kind=user)
	Agents    map[string]string `json:"agents"`     // agent id → xoxb- bot token (posts as that agent)
}

// LoadConfig resolves config from .hatch/slack/config.json, then lets the
// environment override scalar fields. Per-agent tokens may also be supplied as
// HATCH_SLACK_TOKEN_<ID> (e.g. HATCH_SLACK_TOKEN_CLAUDE_CODE for "claude-code").
func LoadConfig(l paths.Layout) (Config, error) {
	var c Config
	if raw, err := os.ReadFile(l.SlackConfig()); err == nil {
		_ = json.Unmarshal(raw, &c) // best-effort: env still overrides below
	}
	if c.Agents == nil {
		c.Agents = map[string]string{}
	}
	if v := os.Getenv("HATCH_SLACK_APP_TOKEN"); v != "" {
		c.AppToken = v
	}
	if v := os.Getenv("HATCH_SLACK_BOT_TOKEN"); v != "" {
		c.HubToken = v
	}
	if v := os.Getenv("HATCH_SLACK_CHANNEL"); v != "" {
		c.ChannelID = v
	}
	if v := os.Getenv("HATCH_SLACK_BOSS"); v != "" {
		c.Boss = v
	}
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || v == "" || !strings.HasPrefix(k, "HATCH_SLACK_TOKEN_") {
			continue
		}
		id := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(k, "HATCH_SLACK_TOKEN_"), "_", "-"))
		c.Agents[id] = v
	}
	c.AppToken = strings.TrimSpace(c.AppToken)
	c.HubToken = strings.TrimSpace(c.HubToken)
	c.ChannelID = strings.TrimSpace(c.ChannelID)
	c.Boss = strings.TrimSpace(c.Boss)
	return c, c.validate()
}

func (c Config) validate() error {
	var missing []string
	if c.AppToken == "" {
		missing = append(missing, "hub app token (HATCH_SLACK_APP_TOKEN)")
	}
	if c.HubToken == "" {
		missing = append(missing, "hub bot token (HATCH_SLACK_BOT_TOKEN)")
	}
	if c.ChannelID == "" {
		missing = append(missing, "channel id (HATCH_SLACK_CHANNEL)")
	}
	if c.Boss == "" {
		missing = append(missing, "boss member id (HATCH_SLACK_BOSS)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("slack config incomplete: missing %s", strings.Join(missing, ", "))
	}
	return nil
}
