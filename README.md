# telemetry

Noisy Sockets anonymous telemetry API. Why write our own? Because OMG the 
existing projects in this space are an absolute cesspool of complexity and
over-engineering.

## Opting Out

If you don't want to participate in anonymous telemetry, you can opt out by 
setting the `NSH_NO_TELEMETRY` environment variable.

```sh
export NSH_NO_TELEMETRY=1
```