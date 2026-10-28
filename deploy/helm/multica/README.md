# Deployment availability

The backend defaults to `Recreate` for single-pod installations with local
uploads. Kubernetes stops the old backend before starting its replacement,
including startup migrations. Requests during that gap can fail in the ingress
or the frontend API proxy. Shutdown draining alone cannot remove this gap.

The frontend uses `RollingUpdate` with `maxUnavailable: 0` and `maxSurge: 1`.
Its startup and readiness probes request `/login`, which checks the frontend
without depending on the backend. The old frontend keeps serving for ten
seconds in `preStop` while Kubernetes propagates its removal from endpoints.

## Backend rolling updates

Before enabling overlapping backend pods:

- Configure a shared Redis through `REDIS_URL` in `existingSecret` for
  cross-replica realtime and daemon routing.
- Configure S3 uploads and disable the uploads PVC, or use a shared
  `ReadWriteMany` uploads volume. The chart rejects rolling updates with the
  default single-writer volume. Do not disable persistence while keeping
  uploads on pod-local disk.
- Allow enough cluster resources to run one extra backend and frontend pod.
- Use database migrations compatible with both application versions. Startup
  migrations are serialized by a PostgreSQL advisory lock; serialization does
  not make destructive schema changes compatible with the old backend.

After those prerequisites are in place, merge this into the installation's
values file (replace the bucket with the configured bucket):

```yaml
backend:
  replicas: 2
  strategy: RollingUpdate
  terminationGracePeriodSeconds: 120
  uploads:
    persistence:
      enabled: false
  config:
    s3Bucket: your-multica-uploads
    shutdownHoldDuration: 10s
```

The backend rollout keeps all desired replicas available and starts one extra
pod. A replacement receives traffic only after `/healthz` confirms database
connectivity and the complete migration set. A startup failure leaves the old
pods available instead of removing them first.

On SIGTERM or SIGINT, the backend immediately makes `/readyz` and `/healthz`
return HTTP 503, bypassing cached readiness. `/health` continues to return 200.
During `shutdownHoldDuration`, ordinary requests still run and dependencies
remain available. After the hold, the existing HTTP and worker shutdown
sequence runs. A second signal interrupts the hold.

Load balancers must use `/readyz` or `/healthz` for backend readiness. Tune the
hold to exceed their probe and endpoint propagation delay; ten seconds is a
starting point, not a guarantee for every load balancer. Keep
`terminationGracePeriodSeconds` longer than the hold plus request and worker
teardown. The binary's default hold remains zero; the chart also defaults to
`0s` to preserve immediate shutdown for `Recreate` installations.

Render and validate the merged values locally before upgrading. During an
approved rollout, check that the new pod becomes ready before the old pod
exits, and monitor requests through both the frontend proxy and API ingress.
Existing WebSocket connections reconnect when their pod exits.

## Rollback

Roll back the application and chart to a compatible release using the normal
Helm rollback procedure. This does not reverse database migrations, so verify
the old binary's schema compatibility first. Returning to `Recreate` restores
its maintenance gap; retain shared upload storage and Redis while any
overlapping pods remain. Do not turn off S3 or remove the shared volume during
the same rollback.
