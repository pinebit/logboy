# MVP Scope

1. OBRY service with proper CI/CD, published as a docker image.
2. Landing page.
3. Online documentation.
4. Online wizard for docker-compose?

-----

1. ORBY service features
- shall support multiple RPCs, one per network
- shall support multiple contracts & events
- config with yaml/toml
- pulling logs and handle reorgs
- plugins framework
    - simple Prometheus exporter
    - Postgres ingestor
    - log writer (zap)
- CI/CD
- overall quality & tests

Not for MVP:
- built-in visualization
- built-in alerting
