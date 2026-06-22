package slack

import (
	"fmt"
	"io"
	"os"
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
// In the default mode it blocks on the Socket Mode connection (IN) while a
// ticker mirrors the bus (OUT); --once and --dry-run are non-blocking.
func Run(l paths.Layout, o Options) error {
	if o.Stdout == nil {
		o.Stdout = os.Stdout
	}
	b := bus.New(l)
	rs := roster.New(l)
	tm := loadThreadmap(l)

	if o.DryRun {
		cfg := Config{Boss: os.Getenv("HATCH_SLACK_BOSS"), ChannelID: os.Getenv("HATCH_SLACK_CHANNEL")}
		br := NewBridge(b, rs, cfg, &dryPoster{w: o.Stdout}, tm)
		fmt.Fprintln(o.Stdout, "— slack bridge dry-run: mirroring current bus backlog —")
		return br.mirrorOnce(time.Now())
	}

	cfg, err := LoadConfig(l)
	if err != nil {
		return err
	}
	api := slackapi.New(cfg.BotToken, slackapi.OptionAppLevelToken(cfg.AppToken))
	br := NewBridge(b, rs, cfg, &realPoster{api: api, channel: cfg.ChannelID}, tm)

	if o.Once {
		return br.mirrorOnce(time.Now())
	}

	sm := socketmode.New(api)
	go consume(sm, br)
	go func() {
		tk := time.NewTicker(o.Interval)
		defer tk.Stop()
		for range tk.C {
			if err := br.mirrorOnce(time.Now()); err != nil {
				fmt.Fprintf(o.Stdout, "slack mirror error: %v\n", err)
			}
		}
	}()
	fmt.Fprintf(o.Stdout, "— slack bridge live on channel %s (boss=%s) —\n", cfg.ChannelID, cfg.Boss)
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

// realPoster posts to Slack as the impersonated agent. Username/icon override
// needs the bot scope chat:write.customize.
type realPoster struct {
	api     *slackapi.Client
	channel string
}

func (p *realPoster) post(threadTS, username, icon, text string) (string, error) {
	opts := []slackapi.MsgOption{
		slackapi.MsgOptionText(text, false),
		slackapi.MsgOptionUsername(username),
		slackapi.MsgOptionIconEmoji(icon),
		slackapi.MsgOptionAsUser(false),
	}
	if threadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(threadTS))
	}
	_, ts, err := p.api.PostMessage(p.channel, opts...)
	return ts, err
}

// dryPoster prints what would be sent, so the OUT path can be smoke-tested
// without any Slack credentials. It fabricates monotonic ts values so the
// threadmap behaves as it would live.
type dryPoster struct {
	w io.Writer
	n int
}

func (p *dryPoster) post(threadTS, username, icon, text string) (string, error) {
	where := "(new thread)"
	if threadTS != "" {
		where = "↳ " + threadTS
	}
	fmt.Fprintf(p.w, "  %s %-12s %s\n      %s\n", icon, username, where, text)
	p.n++
	return fmt.Sprintf("dry-%d", p.n), nil
}
