package dice

// ServeQQ is retained as the compatibility hook used by built-in QQ process
// helpers. Normal endpoint startup must enter through StartEndpointLifecycle.
// When a built-in process becomes ready, this function performs one protocol
// websocket attempt inside the already active supervisor generation.
func ServeQQ(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep == nil || ep.Platform != "QQ" {
		return
	}
	ep.BindRuntime(d.ImSession)

	if ep.lifecycle == nil || ep.State != StateConnecting {
		_ = StartEndpointLifecycle(d, ep)
		return
	}

	startQQProtocolConnection(d, ep)
}

// startQQProtocolConnection performs exactly one protocol connection attempt
// after a built-in QQ process has exposed its websocket endpoint.
func startQQProtocolConnection(d *Dice, ep *EndPointInfo) {
	if ep == nil || ep.Adapter == nil {
		return
	}
	ep.BindRuntime(d.ImSession)
	switch conn := ep.Adapter.(type) {
	case *PlatformAdapterGocq:
		_ = conn.Serve()
	case *PlatformAdapterWalleQ:
		_ = conn.Serve()
	}
}
