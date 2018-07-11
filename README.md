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
## Test the actuator standalone
Run the expected prebuild environment:
```
create an aws iam profile/role named openshift_node_describe_instances
create an aws iam profile/role named openshift_master_launch_instances
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

## Test the actuator along with the machine controller and the cluster API on minikube
Run the expected prebuild environment as above. Then:
```
minikube start
eval $(minikube docker-env)
```
Change source code and:
```
make aws-machine-controller-image
```
Add your aws credentials to the addons.yaml file in base64 format:
```
echo -n 'your_id' | base64
echo -n 'your_key' | base64
```
Deploy the comoponents:
```
kubectl apply examples/addons.yaml
kubectl apply examples/cluster-api-server.yaml
kubectl apply examples/provider-components.yml
```
Deploy the machines:
```
kubectl apply examples/machine.yaml --validate=false
```
or alternatively:
```
kubectl apply examples/machine-set.yaml --validate=false
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
