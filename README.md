# Cluster API Provider AWS

## Build the controller
Update dependencies if needed:
```
glide install --strip-vendor
glide-vc --use-lock-file --no-tests --only-code
```
Then:

```
cd cmd
go build
```
## Test the actuator
Run the expected prebuild environment:
```
create an aws iam profile/role named openshift_node_describe_instances
create an aws ssh key named actuator
cd aws-actuator-test/prebuild
terraform apply
```
Then:
```
cd aws-actuator-test
go build
./aws-actuator-test create test
./aws-actuator-test exists test
./aws-actuator-test delete test
```
## Run it with Cluster API clusterctl
Include the aws provider https://github.com/enxebre/cluster-api/blob/master/clusterctl/cmd/create_cluster.go#L160
```
dep ensure
cd clusterctl
```
Follow the docs https://github.com/enxebre/cluster-api/tree/master/clusterctl
```
go build
./clusterctl create cluster --provider aws -c cluster.yaml -m machines.yaml -p provider-components.yaml -a addons.yaml
```
