package termui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/nishchay-veer/gladiator/internal/config"
	"github.com/nishchay-veer/gladiator/internal/game"
	"github.com/nishchay-veer/gladiator/internal/netplay"
)

type PlayLocalOptions struct {
	Config config.Config
}

type PlayHostOptions struct {
	Config         config.Config
	Addr           string
	LinkSimulation netplay.LinkSimulation
}

type PlayJoinOptions struct {
	Config         config.Config
	HostAddr       string
	PlayerName     string
	BuildVersion   string
	JoinTimeout    time.Duration
	LinkSimulation netplay.LinkSimulation
}

func PlayLocal(ctx context.Context, opts PlayLocalOptions) error {
	cfg := opts.Config
	if cfg.SimulationRate <= 0 {
		cfg = config.Default()
	}

	state, err := game.NewLocalState()
	if err != nil {
		return err
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	screen.EnableMouse()
	screen.HideCursor()

	app := localApp{
		screen:      screen,
		cfg:         cfg,
		state:       state,
		events:      make(chan tcell.Event, 32),
		player:      game.PlayerOne,
		pending:     game.NewInputCommand(state.Tick, game.PlayerOne),
		showPlayer2: true,
	}

	go app.pollEvents()
	return app.run(ctx)
}

func PlayHost(ctx context.Context, opts PlayHostOptions) error {
	cfg := opts.Config
	if cfg.SimulationRate <= 0 {
		cfg = config.Default()
	}

	host, err := netplay.ListenHost(netplay.HostOptions{Addr: opts.Addr})
	if err != nil {
		return err
	}
	defer host.Close()

	state, err := game.NewLocalState()
	if err != nil {
		return err
	}
	state = applySnapshot(state, host.Snapshot())

	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	screen.EnableMouse()
	screen.HideCursor()

	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	session := host.StartSession(sessionCtx, netplay.SessionOptions{
		SimulationRate: cfg.SimulationRate,
		SnapshotRate:   cfg.SnapshotRate,
		LinkSimulation: opts.LinkSimulation,
	})

	app := localApp{
		screen:      screen,
		cfg:         cfg,
		state:       state,
		events:      make(chan tcell.Event, 32),
		player:      game.PlayerOne,
		status:      peerStatusText(netplay.PeerStatus{}),
		pending:     game.NewInputCommand(state.Tick, game.PlayerOne),
		showPlayer2: false,
	}

	go app.pollEvents()
	return app.runHost(ctx, session)
}

func PlayJoin(ctx context.Context, opts PlayJoinOptions) error {
	cfg := opts.Config
	if cfg.SimulationRate <= 0 {
		cfg = config.Default()
	}

	client, err := netplay.DialClient(netplay.ClientOptions{
		HostAddr:     opts.HostAddr,
		PlayerName:   opts.PlayerName,
		BuildVersion: opts.BuildVersion,
	})
	if err != nil {
		return err
	}
	defer client.Close()

	joinTimeout := opts.JoinTimeout
	if joinTimeout <= 0 {
		joinTimeout = 5 * time.Second
	}
	joinCtx, cancelJoin := context.WithTimeout(ctx, joinTimeout)
	welcome, err := client.Join(joinCtx)
	cancelJoin()
	if err != nil {
		return err
	}

	state, err := game.NewLocalState()
	if err != nil {
		return err
	}
	state = applySnapshot(state, welcome.Snapshot)

	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	screen.EnableMouse()
	screen.HideCursor()

	sessionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	session := client.StartJoinedSession(sessionCtx, welcome, netplay.SessionOptions{
		SimulationRate: cfg.SimulationRate,
		SnapshotRate:   cfg.SnapshotRate,
		LinkSimulation: opts.LinkSimulation,
	})

	app := localApp{
		screen:      screen,
		cfg:         cfg,
		state:       state,
		events:      make(chan tcell.Event, 32),
		player:      welcome.PlayerID,
		status:      "LAN P2",
		pending:     game.NewInputCommand(state.Tick, welcome.PlayerID),
		prediction:  game.NewPredictionHistory(0),
		showPlayer2: true,
	}

	go app.pollEvents()
	err = app.runClient(ctx, session)
	cancel()
	disconnectCtx, cancelDisconnect := context.WithTimeout(context.Background(), 250*time.Millisecond)
	_ = client.Disconnect(disconnectCtx, "quit")
	cancelDisconnect()
	return err
}

type localApp struct {
	screen       tcell.Screen
	cfg          config.Config
	state        game.State
	events       chan tcell.Event
	player       game.PlayerID
	status       string
	pending      game.InputCommand
	showPlayer2  bool
	showNetDebug bool
	netDebug     netDebugStats
	prediction   *game.PredictionHistory
	correction   correctionAnimation

	fpsWindow time.Time
	frames    int
	fps       int
}

func (a *localApp) pollEvents() {
	for {
		event := a.screen.PollEvent()
		if event == nil {
			close(a.events)
			return
		}
		a.events <- event
	}
}

func (a *localApp) run(ctx context.Context) error {
	ticker := time.NewTicker(a.cfg.SimulationRate)
	defer ticker.Stop()

	a.draw()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-a.events:
			if !ok {
				return nil
			}
			if quit := a.handleEvent(event); quit {
				return nil
			}
			a.draw()
		case <-ticker.C:
			frame := game.NewInputFrame(a.state.Tick)
			frame.Set(a.consumePendingCommand())
			frame.Set(a.state.BotCommand(game.PlayerTwo, game.PlayerOne, a.state.Tick))
			a.state.Step(frame)
			a.draw()
		}
	}
}

func (a *localApp) runHost(ctx context.Context, session netplay.HostSession) error {
	return a.runSession(ctx, session.Inputs, session.Snapshots, session.PeerStatus, session.Errors, netDebugSource{
		NetworkStats: session.NetworkStats,
		LinkStats:    session.LinkStats,
	})
}

func (a *localApp) runClient(ctx context.Context, session netplay.ClientSession) error {
	return a.runSession(ctx, session.Inputs, session.Snapshots, nil, session.Errors, netDebugSource{
		NetworkStats: session.NetworkStats,
		LinkStats:    session.LinkStats,
	})
}

func (a *localApp) runSession(ctx context.Context, inputs chan<- game.InputCommand, snapshots <-chan game.Snapshot, peerStatus <-chan netplay.PeerStatus, errors <-chan error, debugSource netDebugSource) error {
	inputTicker := time.NewTicker(a.cfg.SimulationRate)
	defer inputTicker.Stop()

	renderTicker := time.NewTicker(renderRate(a.cfg))
	defer renderTicker.Stop()

	a.updateNetDebug(debugSource)
	a.draw()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errors:
			return err
		case snapshot := <-snapshots:
			a.applyAuthoritativeSnapshot(snapshot)
			a.updateNetDebug(debugSource)
		case status, ok := <-peerStatus:
			if !ok {
				peerStatus = nil
				continue
			}
			a.applyPeerStatus(status)
		case event, ok := <-a.events:
			if !ok {
				return nil
			}
			if quit := a.handleEvent(event); quit {
				return nil
			}
			a.updateNetDebug(debugSource)
		case <-inputTicker.C:
			command, sent := a.sendPendingCommand(inputs)
			if sent {
				a.applyPredictedCommand(command)
			}
			a.correction.Advance()
			a.updateNetDebug(debugSource)
		case <-renderTicker.C:
			a.updateNetDebug(debugSource)
			a.draw()
		}
	}
}

func renderRate(cfg config.Config) time.Duration {
	if cfg.RenderRate > 0 {
		return cfg.RenderRate
	}
	if cfg.SimulationRate > 0 {
		return cfg.SimulationRate
	}
	return config.Default().RenderRate
}

func (a *localApp) updateNetDebug(source netDebugSource) {
	a.netDebug = source.snapshot()
}

func (a *localApp) applyPeerStatus(status netplay.PeerStatus) {
	a.status = peerStatusText(status)
	a.showPlayer2 = status.Connected
}

func (a *localApp) applyAuthoritativeSnapshot(snapshot game.Snapshot) {
	if a.prediction == nil {
		a.state = applySnapshot(a.state, snapshot)
		return
	}

	before := a.inputPlayer().Position
	result := a.prediction.Reconcile(a.state, snapshot, a.player)
	a.state = result.State
	after := a.inputPlayer().Position
	if result.NeedsCorrection {
		a.correction = newCorrectionAnimation(a.player, before, after)
	}
}

func (a *localApp) applyPredictedCommand(command game.InputCommand) {
	if a.prediction == nil || command.PlayerID != a.player {
		return
	}

	command.Tick = a.state.Tick
	frame := game.NewInputFrame(a.state.Tick)
	frame.Set(command)
	a.state.Step(frame)
	a.prediction.Record(command, a.state.Snapshot())
	a.correction.Retarget(a.inputPlayer().Position)
}

func peerStatusText(status netplay.PeerStatus) string {
	if status.Connected {
		return "P2 LIVE"
	}
	if status.Reason == "timeout" {
		return "P2 LOST"
	}
	return "P2 WAIT"
}
