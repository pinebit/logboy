# LOGNITE. ROADMAP.

Version 0.1 "MVP"
+ Ability to connect to multiple blockchains concurrently
+ Ability to watch for multiple contracts concurrently
+ Outputs: logger, prometheus, postgres, csv?
+ Handling reorgs and rpc reconnections (backfilling)
+ Separate service logging and events logging
+ Hosted demo instances with Grafana monitoring/alerting
+ Load testing passed

MVP public appearance:
+ Landing page + io domain
+ Intro video and infographics
+ Public docker image
+ GitHub with tutorials and basic documentation
+ Promotion

Version 0.2 "MVP+"
+ Tested with many contracts/chains and all bugs fixed
+ Hosted technical documentation

Version 0.3 "Built-in Status Page"
+ Embed tiny react app to render real-time service status
+ Basic alerting

Version 0.5 "Outputs+"
+ Add InfluxDB, Kafka and other outputs

From now on, moving towards all-in-one solution.

Version 0.7 "Grafana"
+ Semi/Auto-generated dashboards

Version 1.x "Chains+"
+ Support non-EVM blockchains

Version 2.x "Lognite Cloud"
+ Web portal to manage everything in one place
+ Infrastructure to spawn lognite instances
+ Configuration with UI wizards
+ Integrations with Grafana-like clouds
