#!/bin/bash
# $1 num of dumps
# $2 OpenLambda path
# $3 machines list

ARR=("$@")

num_dumps=${ARR[0]}
echo "number of lambda dumps: $num_dumps"

# openlambda_path="/mnt/lambda_scheduler/s19-lambda"
openlambda_path=${ARR[1]}
echo "OpenLambda path: $openlambda_path"

#machines=(
#	'c220g2-011027.wisc.cloudlab.us'
#	'c220g2-011025.wisc.cloudlab.us'
#	'c220g2-011026.wisc.cloudlab.us'
#	)
(( num_of_machines = ${#ARR[@]} - 2 ))
echo "number of machines: $num_of_machines"

for i in $( eval echo {1..$num_of_machines}); do
	(( index = $i + 1 )) 
	machine=${ARR[$index]}
	echo "initializing machine: $machine"
	#stop workers
	ssh root@${machine} "cd ${openlambda_path}; ./bin/admin kill --cluster=my-cluster"
	ssh root@${machine} "rm -r ${openlambda_path}/my-cluster"
	ssh root@${machine} "kill -9 $(lsof -t -i:8081)"
	ssh root@${machine} "kill -9 $(lsof -t -i:8081)"

 	#reinitialize workers
	ssh root@${machine} "make -C $openlambda_path clean"
	ssh root@${machine} "make -C $openlambda_path"
	ssh root@${machine} "cd ${openlambda_path}; ./bin/admin new --cluster=my-cluster"
	ssh root@${machine} "pip2 install -r /mnt/pipbench/requirements.txt  --index-url http://localhost:9199/simple/ --target=/mnt/lambda_scheduler/s19-lambda/my-cluster/base/packages"
	scp ${openlambda_path}/benchmark/template.json root@${machine}:${openlambda_path}/my-cluster/config/template.json

	# dump lambda code
	ssh root@${machine} "cp ${openlambda_path}/benchmark/handlers/handlers.tar ${openlambda_path}/my-cluster/registry/; cd ${openlambda_path}/my-cluster/registry/; tar xf ${openlambda_path}/my-cluster/registry/handlers.tar; rm ${openlambda_path}/my-cluster/registry/handlers.tar"

	#start workers
	ssh root@${machine} "cd ${openlambda_path}; ./bin/admin workers --cluster=my-cluster --port=8081"
	ssh root@${machine} "cd ${openlambda_path}; ./bin/admin status --cluster=my-cluster"
done

now=$(date +"%T")
echo "Current time : $now"

CUR_DIR=$(cd $(dirname $0); pwd)
BENCHMARK_PARTH="$openlambda_path/benchmark"
echo $BENCHMARK_PARTH

cd $openlambda_path
./bin/admin kill --cluster=my-cluster
#rm -r my-cluster
kill -9 $(lsof -t -i:8079)
rm -rf my-cluster
./bin/admin new --cluster=my-cluster
cp $BENCHMARK_PARTH/lard_load_balancer.json my-cluster/config/load_balancer.json

cp ${openlambda_path}/benchmark/handlers/handlers.tar ${openlambda_path}/my-cluster/registry/
cd ${openlambda_path}/my-cluster/registry/ 
tar xf ${openlambda_path}/my-cluster/registry/handlers.tar
rm ${openlambda_path}/my-cluster/registry/handlers.tar

cd ${openlambda_path}
./bin/admin load-balancer --cluster=my-cluster
./bin/admin status --cluster=my-cluster
