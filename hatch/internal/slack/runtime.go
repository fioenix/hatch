package slack

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/fioenix/overclaud/hatch/internal/bus"
	"github.com/fioenix/overclaud/hatch/internal/paths"
	"github.com/fioenix/overclaud/hatch/internal/roster"
	slackapi "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// Options configures a bridge run.
type Options struct {
	Interval time.Duration // OUT mirror poll period
	Once     bool          // mirror current backlog once and exit (no Socket Mode)
	DryRun   bool          // print OUT to Stdout instead of Slack; needs no tokens
	Stdout   io.Writer
}

// Run assembles and runs the bridge for a workspace. It keeps all slack-go
// dependencies inside this package so the rest of the binary stays Slack-free.
// In the default mode it blocks on the hub app's Socket Mode connection (IN)
// while a ticker mirrors the bus (OUT); --once and --dry-run are non-blocking.
func Run(l paths.Layout, o Options) error {
	if o.Stdout == nil {
		o.Stdout = os.Stdout
	}
	b := bus.New(l)
	rs := roster.New(l)
	tm := loadThreadmap(l)

	if o.DryRun {
		cfg := Config{Boss: os.Getenv("HATCH_SLACK_BOSS"), ChannelID: os.Getenv("HATCH_SLACK_CHANNEL")}
		br := NewBridge(b, rs, cfg, &dryPoster{w: o.Stdout}, newMemThreadmap(), nil)
		fmt.Fprintln(o.Stdout, "— slack bridge dry-run: mirroring current bus backlog —")
		return br.mirrorOnce(time.Now())
	}

	cfg, err := LoadConfig(l)
	if err != nil {
		return err
	}

	hub := slackapi.New(cfg.HubToken, slackapi.OptionAppLevelToken(cfg.AppToken))
	mp := &multiPoster{channel: cfg.ChannelID, hub: hub, agents: map[string]*slackapi.Client{}}
	mentions := map[string]string{} // slack bot user-id → agent id
	for _, id := range sortedKeys(cfg.Agents) {
		cl := slackapi.New(cfg.Agents[id])
		mp.agents[id] = cl
		if at, aerr := cl.AuthTest(); aerr == nil && at.UserID != "" {
			mentions[at.UserID] = id
			fmt.Fprintf(o.Stdout, "slack: %s → bot %s (%s)\n", id, at.User, at.UserID)
		} else if aerr != nil {
			fmt.Fprintf(o.Stdout, "slack: agent %s auth.test failed (will impersonate via hub): %v\n", id, aerr)
		}
	}
	br := NewBridge(b, rs, cfg, mp, tm, mentions)

	if o.Once {
		return br.mirrorOnce(time.Now())
	}

	sm := socketmode.New(hub)
	go consume(sm, br)
	go func() {
		tk := time.NewTicker(o.Interval)
		defer tk.Stop()
		for range tk.C {
			if merr := br.mirrorOnce(time.Now()); merr != nil {
				fmt.Fprintf(o.Stdout, "slack mirror error: %v\n", merr)
			}
		}
	}()
	fmt.Fprintf(o.Stdout, "— slack bridge live on channel %s (boss=%s, %d agent bots) —\n",
		cfg.ChannelID, cfg.Boss, len(mp.agents))
	return sm.Run()
}

// consume reads Socket Mode events, acks them, and feeds genuine messages to
// the IN path. Non-message events are acked and ignored.
func consume(sm *socketmode.Client, br *Bridge) {
	for evt := range sm.Events {
		if evt.Request != nil {
			_ = sm.Ack(*evt.Request)
		}
		if evt.Type != socketmode.EventTypeEventsAPI {
			continue
		}
		api, ok := evt.Data.(slackevents.EventsAPIEvent)
		if !ok || api.Type != slackevents.CallbackEvent {
			continue
		}
		me, ok := api.InnerEvent.Data.(*slackevents.MessageEvent)
		if !ok {
			continue
		}
		_ = br.handleIncoming(incoming{
			ChannelID: me.Channel,
			User:      me.User,
			BotID:     me.BotID,
			SubType:   me.SubType,
			ThreadTS:  me.ThreadTimeStamp,
			TS:        me.TimeStamp,
			Text:      me.Text,
		})
	}
}

// multiPoster posts as each agent's own Slack bot. An agent with a configured
// client posts under its real identity; anyone else (e.g. the "hatch" voice for
// escalations, or an agent without a token) falls back to the hub bot with a
// username/icon override.
type multiPoster struct {
	channel string
	hub     *slackapi.Client
	agents  map[string]*slackapi.Client
}

func (p *multiPoster) post(from, threadTS, displayName, icon, text string) (string, error) {
	if cl, ok := p.agents[from]; ok {
		return sendMsg(cl, p.channel, threadTS, "", "", text) // real bot: its own name/avatar
	}
	return sendMsg(p.hub, p.channel, threadTS, displayName, icon, text) // impersonation fallback
}

// sendMsg posts text to a channel, optionally as a thread reply and optionally
// impersonating a username+icon (only when username is non-empty).
func sendMsg(cl *slackapi.Client, channel, threadTS, username, icon, text string) (string, error) {
	opts := []slackapi.MsgOption{slackapi.MsgOptionText(text, false)}
	if username != "" {
		opts = append(opts, slackapi.MsgOptionUsername(username), slackapi.MsgOptionIconEmoji(icon), slackapi.MsgOptionAsUser(false))
	}
	if threadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(threadTS))
	}
	_, ts, err := cl.PostMessage(channel, opts...)
	return ts, err
}

// dryPoster prints what would be sent, so the OUT path can be smoke-tested
// without any Slack credentials. It fabricates monotonic ts values so the
// threadmap behaves as it would live.
type dryPoster struct {
	w io.Writer
	n int
}

func (p *dryPoster) post(from, threadTS, displayName, icon, text string) (string, error) {
	where := "(new thread)"
	if threadTS != "" {
		where = "↳ " + threadTS
	}
	fmt.Fprintf(p.w, "  %s %-12s %s\n      %s\n", icon, from, where, text)
	p.n++
	return fmt.Sprintf("dry-%d", p.n), nil
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
