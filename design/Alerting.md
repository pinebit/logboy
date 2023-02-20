# Implementing Custom Alerts from Events

This module can be used if Grafana is not used.

A user could specify mask/regexp for an event:
```yaml
- FooBar:
    - SomeEvent:
        - user:
            oneof:
                -0x123...
                -0x456...
        - foo:
            lt: 5
            gt: 10
```

Desired use cases:
- if a certain event is emited
- if a certain event is emited with certain params
- if an event is emited before/after another event

```yaml
- Alert:
    Name: "Request has been fulfilled without a request"
    Type: Order
    EventA: FooBar.Request
    EventB: FooBar.Fulfilled
    Match: FooBar.Request.requestID == FooBar.Fulfilled.requestID
```

- if expected event B is not emited within timeout after event A

```yaml
- Alert:
    Name: "Request has not been fulfilled within 5 min"
    Type: Timeout
    Initiator: FooBar.Request
    Finisher: FooBar.Fulfilled
    Match: FooBar.Request.requestID == FooBar.Fulfilled.requestID
    Timeout: 5min
```

Once a condition is met, the alert can be sent as:
- logged as desired log-level, e.g. `error`
- call a webhook, e.g. PagerDuty
- built-in Telegram bot (a plugin?)

