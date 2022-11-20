# Minikom

## Run it
1. `docker compose build`
1. `docker compose up`


```yaml
# Accept a single event from Komobox
Method: POST
Endpoint: /event
Request type: JSON object
Request structure: {
  "service_name": string
  "state": "deploy" or "issue"
  "timestamp": number (unix time) # the time the event took place at the source
}
Response: 200 OK # No body
```

```yaml
# All services that have been seen by Minikom so far
Method: GET
Endpoint: /services
Response type: JSON object
Response structure: {
  <service_name>: {
    state: "idle" or "deploy" or "issue" # latest state of the service
    since: number (unix time) # the time the latest state started
  }
}
```

```yaml
# List latest states (max: 50) for service
Method: GET
Endpoint: /services/<service_name>/latest-states
Response type: JSON array
Response structure: [{ # element per state change
   state: "deploy" or "issue" # the state type (idle should not be included)
   start_time: number (unix time) # the time the state started
   end_time: number (unix time) or null # the time the state ended (null if still ongoing)
}]
```
