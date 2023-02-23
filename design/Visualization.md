# How To Visualize Events

This is a built-in web-app that is a simple alternative to Grafana.

An uber goal would be to run LOGNITE alone without any external dependency, such as Grafana, Postgres, etc.
To do that, we need a local DB to persist time-series, metrics, alerts, logs, etc.

Visualizer:
- a built-in webapp immediately ready to use at a certain port.
- displays all monitoring contracts and RPCs.
- displays all prom counters per contract.
- the home dashboard shows all pinned panels.
- displays real-time logs.
- displays events data as tables.
- can render a graph/histogram/etc for an argument value (time-series).
- displays all alerts, can mute/unmute alerts...

A comprehensive app should manage everything from web-ui...
but this is toooo complicated to build. a replacement to all grafana, does not make sense.

The built-in version must be a simple read-only tool.
Maybe only Home Dashboard can persist its pinned items.