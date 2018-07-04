# Cluster API Provier AWS

## Build
Update dependencies if needed:

`glide install --strip-vendor`

`glide-vc --use-lock-file --no-tests --only-code`

Then:

`cd cmd`

`go build`

## Run with Cluster API clusterctl
Include the aws provider https://github.com/enxebre/cluster-api/blob/master/clusterctl/cmd/create_cluster.go#L160

`dep ensure`

`cd clusterctl`

Follow the docs https://github.com/enxebre/cluster-api/tree/master/clusterctl

`go build`

`./clusterctl create cluster --provider aws -c cluster.yaml -m machines.yaml -p provider-components.yaml -a addons.yaml`
