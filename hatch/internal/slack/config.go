// Package slack bridges the Hatch chat bus to a real Slack channel: it mirrors
// every squad message into Slack (each agent impersonated by name + icon) and
// ingests the human boss's Slack messages back onto the bus, where the wake
// daemon delivers them. It is a presentation/ingress adapter — it never
// orchestrates work; the boss in Slack is just a peer who happens to sit
// outside the terminal.
package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/paths"
)

// Config holds the credentials and routing the bridge needs. Tokens are the
// boss's own Slack app secrets — loaded from the environment first, then a
// gitignored .hatch/slack/config.json. They are never committed.
type Config struct {
	BotToken  string `json:"bot_token"`  // xoxb- (chat:write)
	AppToken  string `json:"app_token"`  // xapp- (connections:write, Socket Mode)
	ChannelID string `json:"channel_id"` // the #squad channel (C...)
	Boss      string `json:"boss"`       // member id of the human (Kind=user)
}

// LoadConfig resolves config from the environment, falling back per-field to
// .hatch/slack/config.json. Env wins so secrets can stay out of files.
func LoadConfig(l paths.Layout) (Config, error) {
	var c Config
	if raw, err := os.ReadFile(l.SlackConfig()); err == nil {
		_ = json.Unmarshal(raw, &c) // best-effort: env still overrides below
	}
	if v := os.Getenv("HATCH_SLACK_BOT_TOKEN"); v != "" {
		c.BotToken = v
	}
	if v := os.Getenv("HATCH_SLACK_APP_TOKEN"); v != "" {
		c.AppToken = v
	}
	if v := os.Getenv("HATCH_SLACK_CHANNEL"); v != "" {
		c.ChannelID = v
	}
	if v := os.Getenv("HATCH_SLACK_BOSS"); v != "" {
		c.Boss = v
	}
	c.BotToken = strings.TrimSpace(c.BotToken)
	c.AppToken = strings.TrimSpace(c.AppToken)
	c.ChannelID = strings.TrimSpace(c.ChannelID)
	c.Boss = strings.TrimSpace(c.Boss)
	return c, c.validate()
}

func (c Config) validate() error {
	var missing []string
	if c.BotToken == "" {
		missing = append(missing, "bot token (HATCH_SLACK_BOT_TOKEN)")
	}
	if c.AppToken == "" {
		missing = append(missing, "app token (HATCH_SLACK_APP_TOKEN)")
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
