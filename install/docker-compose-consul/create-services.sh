#!/bin/zsh

curl --request PUT --data-binary @./data/gateways/gloo-system/gw-proxy.yaml http://127.0.0.1:8500/v1/kv/gloo/gateway.solo.io/v1/Gateway/gloo-system


PETSTORE_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' docker-compose-consul_petstore_1)
cat > petstore-service.json <<EOF
{
  "ID": "petstore1",
  "Name": "petstore",
  "Address": "${PETSTORE_IP}",
  "Port": 8080
}
EOF
curl -v \
    -XPUT \
    --data @petstore-service.json \
    "http://127.0.0.1:8500/v1/agent/service/register"

# create ExtAuth Service
EXTAUTH_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' docker-compose-consul_extauth_1)
cat > extauth-service.json <<EOF
{
  "ID": "extauth1",
  "Name": "extauth",
  "Address": "${EXTAUTH_IP}",
  "Port": 8083
}
EOF
curl -v \
    -XPUT \
    --data @extauth-service.json \
    "http://127.0.0.1:8500/v1/agent/service/register"

# add http2 flag to ExtAuth upstream
curl -X PUT --data-binary @./extauth.yaml http://127.0.0.1:8500/v1/kv/gloo/gloo.solo.io/v1/Upstream/gloo-system/extauth

# upload OPA .rego policy as generic Artifact (eqv. Kubernetes ConfigMap)
curl -X PUT --data-binary @./opa-policy.yaml http://127.0.0.1:8500/v1/kv/gloo/gloo.solo.io/v1/Artifact/gloo-system

# upload AuthConfig
curl --request PUT --data-binary @./auth-config.yaml http://127.0.0.1:8500/v1/kv/gloo/enterprise.gloo.solo.io/v1/AuthConfig/gloo-system

# create VirtualService
glooctl add route \
    --path-exact /all-pets \
    --dest-name petstore \
    --prefix-rewrite /api/pets \
    --use-consul


