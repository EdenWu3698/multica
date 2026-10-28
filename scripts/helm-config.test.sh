#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHART_DIR="$ROOT_DIR/deploy/helm/multica"

require_rendered_value() {
  local rendered=$1
  local expected=$2

  if ! grep -Fq "$expected" <<<"$rendered"; then
    echo "Missing expected Helm-rendered config value:"
    echo "  $expected"
    exit 1
  fi
}

reject_rendered_value() {
  local rendered=$1
  local forbidden=$2

  if grep -Fq "$forbidden" <<<"$rendered"; then
    echo "Forbidden Helm-rendered config value:"
    echo "  $forbidden"
    exit 1
  fi
}

helm lint "$CHART_DIR"

default_config="$(
  helm template multica "$CHART_DIR" \
    --show-only templates/configmap.yaml
)"
require_rendered_value "$default_config" 'MULTICA_VCS_INTEGRATION_ENABLED: "true"'
require_rendered_value "$default_config" 'MULTICA_CLOUD_URL: ""'
require_rendered_value "$default_config" 'MULTICA_DATABASE_STARTUP_TIMEOUT: "3m"'
require_rendered_value "$default_config" 'MULTICA_DATABASE_CONNECT_TIMEOUT: "5s"'
require_rendered_value "$default_config" 'MULTICA_SHUTDOWN_HOLD_DURATION: "0s"'

require_rendered_value "$default_config" 'MAINTENANCE_PORT: ""'
maintenance_config="$(helm template multica "$CHART_DIR" --show-only templates/configmap.yaml --set-string backend.config.maintenancePort=6061)"
require_rendered_value "$maintenance_config" 'MAINTENANCE_PORT: "6061"'
maintenance_backend="$(helm template multica "$CHART_DIR" --show-only templates/backend.yaml --set-string backend.config.maintenancePort=6061)"
reject_rendered_value "$maintenance_backend" 'containerPort: 6061'
reject_rendered_value "$maintenance_backend" 'port: 6061'

default_backend="$(
  helm template multica "$CHART_DIR" \
    --show-only templates/backend.yaml
)"
require_rendered_value "$default_backend" 'failureThreshold: 60'
require_rendered_value "$default_backend" 'type: Recreate'
reject_rendered_value "$default_backend" 'rollingUpdate:'
require_rendered_value "$default_backend" 'terminationGracePeriodSeconds: 120'
readiness_block="$(sed -n '/readinessProbe:/,/livenessProbe:/p' <<<"$default_backend")"
require_rendered_value "$readiness_block" 'path: /healthz'
require_rendered_value "$readiness_block" 'periodSeconds: 5'
require_rendered_value "$readiness_block" 'failureThreshold: 1'
liveness_block="$(sed -n '/livenessProbe:/,/resources:/p' <<<"$default_backend")"
require_rendered_value "$liveness_block" 'path: /health'
reject_rendered_value "$liveness_block" 'path: /healthz'
reject_rendered_value "$default_backend" 'SSL_CERT_DIR'
reject_rendered_value "$default_backend" 'extra-ca-certs'

# `helm upgrade --reuse-values` from a chart that predates extraCACerts renders
# without the key at all; that must still render, with no extra trust.
legacy_backend="$(
  helm template multica "$CHART_DIR" \
    --show-only templates/backend.yaml \
    --set backend.extraCACerts=null
)"
reject_rendered_value "$legacy_backend" 'SSL_CERT_DIR'

# Old saved values have no rollout or shutdown keys.
legacy_rollout="$(helm template multica "$CHART_DIR" \
  --set backend.strategy=null \
  --set backend.terminationGracePeriodSeconds=null \
  --set backend.config.shutdownHoldDuration=null)"
require_rendered_value "$legacy_rollout" 'type: Recreate'
require_rendered_value "$legacy_rollout" 'terminationGracePeriodSeconds: 120'
require_rendered_value "$legacy_rollout" 'MULTICA_SHUTDOWN_HOLD_DURATION: "0s"'

rolling_backend="$(helm template multica "$CHART_DIR" \
  --show-only templates/backend.yaml \
  --set backend.strategy=RollingUpdate \
  --set backend.uploads.persistence.enabled=false \
  --set backend.terminationGracePeriodSeconds=150)"
require_rendered_value "$rolling_backend" 'type: RollingUpdate'
require_rendered_value "$rolling_backend" 'maxUnavailable: 0'
require_rendered_value "$rolling_backend" 'maxSurge: 1'
require_rendered_value "$rolling_backend" 'terminationGracePeriodSeconds: 150'

rwx_backend="$(helm template multica "$CHART_DIR" \
  --show-only templates/backend.yaml \
  --set backend.strategy=RollingUpdate \
  --set 'backend.uploads.persistence.accessModes[0]=ReadWriteMany')"
require_rendered_value "$rwx_backend" 'type: RollingUpdate'
require_rendered_value "$rwx_backend" 'ReadWriteMany'

if invalid_backend="$(helm template multica "$CHART_DIR" --set backend.strategy=RollingUpdate 2>&1)"; then
  echo "RollingUpdate unexpectedly accepted a single-writer uploads volume"
  exit 1
fi
require_rendered_value "$invalid_backend" 'requires S3 with uploads.persistence.enabled=false, or a ReadWriteMany uploads volume'

if invalid_backend="$(helm template multica "$CHART_DIR" --set backend.strategy=invalid 2>&1)"; then
  echo "Invalid backend strategy unexpectedly rendered"
  exit 1
fi
require_rendered_value "$invalid_backend" 'backend.strategy must be Recreate or RollingUpdate'

hold_config="$(helm template multica "$CHART_DIR" \
  --show-only templates/configmap.yaml \
  --set-string backend.config.shutdownHoldDuration=10s)"
require_rendered_value "$hold_config" 'MULTICA_SHUTDOWN_HOLD_DURATION: "10s"'

default_frontend="$(helm template multica "$CHART_DIR" --show-only templates/frontend.yaml)"
require_rendered_value "$default_frontend" 'type: RollingUpdate'
require_rendered_value "$default_frontend" 'maxUnavailable: 0'
require_rendered_value "$default_frontend" 'maxSurge: 1'
startup_block="$(sed -n '/startupProbe:/,/readinessProbe:/p' <<<"$default_frontend")"
require_rendered_value "$startup_block" 'path: /login'
require_rendered_value "$startup_block" 'port: http'
frontend_readiness="$(sed -n '/readinessProbe:/,/lifecycle:/p' <<<"$default_frontend")"
require_rendered_value "$frontend_readiness" 'path: /login'
require_rendered_value "$frontend_readiness" 'port: http'
require_rendered_value "$default_frontend" 'command: ["sleep", "10"]'

extra_ca_backend="$(
  helm template multica "$CHART_DIR" \
    --show-only templates/backend.yaml \
    --set backend.extraCACerts.configMap=internal-ca
)"
require_rendered_value "$extra_ca_backend" 'value: /etc/ssl/certs:/etc/multica/ca-certs'
require_rendered_value "$extra_ca_backend" 'mountPath: /etc/multica/ca-certs'
require_rendered_value "$extra_ca_backend" 'name: internal-ca'
require_rendered_value "$extra_ca_backend" 'mountPath: /app/data/uploads'

disabled_config="$(
  helm template multica "$CHART_DIR" \
    --show-only templates/configmap.yaml \
    --set backend.config.vcsIntegrationEnabled=false
)"
require_rendered_value "$disabled_config" 'MULTICA_VCS_INTEGRATION_ENABLED: "false"'

capacity_config="$(
  helm template multica "$CHART_DIR" \
    --show-only templates/configmap.yaml \
    --set-string backend.config.cloud.url=https://multica-cloud.internal
)"
require_rendered_value "$capacity_config" 'MULTICA_CLOUD_URL: "https://multica-cloud.internal"'

echo "helm config rendering ok"
