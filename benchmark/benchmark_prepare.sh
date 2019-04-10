#!/bin/bash
# $1 num of dumps
# $2 OpenLambda path
# $3 machines list

ARR=( "$@" )

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
	stop workers
	ssh root@${machine} "${openlambda_path}/bin/admin kill -cluster=my-cluster"
	ssh root@${machine} "rm -r ${openlambda_path}/my-cluster"
	ssh root@${machine} "kill $(lsof -t -i:8080)"

 	#reinitialize workers
	ssh root@${machine} "make -C $openlambda_path clean"
	ssh root@${machine} "make -C $openlambda_path"
	ssh root@${machine} "cd ${openlambda_path}; ./bin/admin new -cluster my-cluster"
	scp template.json root@${machine}:${openlambda_path}/my-cluster/config/template.json

	# dump lambda code
	for j in $(seq 1 $1);
	do
	        ssh root@${machine} "mkdir ${openlambda_path}/my-cluster/registry/lambda-${j}"
	        scp -r lambda/* root@${machine}:${openlambda_path}/my-cluster/registry/lambda-${j}
	done

	#start workers
	ssh root@${machine} "${openlambda_path}/bin/admin workers -cluster my-cluster"
	ssh root@${machine} "${openlambda_path}/bin/admin status -cluster my-cluster"
done

now=$(date +"%T")
echo "Current time : $now"


