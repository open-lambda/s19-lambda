#!/bin/bash

machines=(
	'c220g2-011027.wisc.cloudlab.us'
	'c220g2-011025.wisc.cloudlab.us'
	'c220g2-011026.wisc.cloudlab.us'
	)

for i in "${machines[@]}"; do
	ssh root@${i} "pip install django"
	ssh root@${i} "pip install pandas"
	ssh root@${i} "pip install matplotlib"
done



