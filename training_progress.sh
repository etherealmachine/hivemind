#!/bin/bash
(
while ((1)); do
	percent=`grep "/" training.log | tail -n 1 | awk 'BEGIN { FS="/" } {print ($1/$2)*100}'`
	echo $percent
	sleep 1
done
) | dialog --shadow --guage "Training Progress" 7 100
