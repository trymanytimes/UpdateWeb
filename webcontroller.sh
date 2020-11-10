#!/bin/sh
#alivetime=$(cat /proc/uptime|awk '{split($1,a,".");print a[1]}')
#if [ $alivetime -le 100 ]; then
#	echo "sleep20"
#	sleep 20
#fi
msg=$(ps -eaf|grep -e "web_controller"|grep -v "grep"|awk '{print $2}')
if [ $1 == "start" ]; then
	if [ "$msg" != "" ]; then
		echo $msg
	else
	containerpg=$(docker ps|grep pg-docker)
	while [ "$containerpg" = "" ]
	do
	echo "docker pg not running." >> path/bin/controller.log
	sleep 1 
	containerpg=$(docker ps|grep pg-docker)
	done
	nohup /root//Applications/Go/github.com/trymanytimes/UpdateWeb/web_controller -c /root//Applications/Go/github.com/trymanytimes/UpdateWeb/etc/web-controller.conf >> /root//Applications/Go/github.com/trymanytimes/UpdateWeb/controller.log 2>&1 &
	fi
else
	if [ "$msg" != "" ]; then
		kill -9 $msg
		echo "web_controller had been killed!"
	else
		echo "not exists!"
	fi
fi
