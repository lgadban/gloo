test-node-port-1                         default  INGRESS    1000      tcp:32412
test-node-port-2                         default  INGRESS    1000      tcp:30832

gcloud compute firewall-rules create test-node-port-2 --allow tcp:30832

open firewalls for nodeport services, gloo-xds and static upstream petstore

