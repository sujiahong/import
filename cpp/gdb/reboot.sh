#!/bin/sh

#
#  $():直接命令行命令执行     ``:当作字符串解析后，再当命令行命令执行
#
#

function exec_name() {
  echo $0 $1
  name4p=`find . -maxdepth 1 -type l -name "*.l" | grep -E "*\.l" | awk -F'/' '{print $NF}' | awk -F. '{print $1}'`
}

function business_name() {
  echo $0 $1
  name4o=`find . -maxdepth 1 -type l -name "libBusiness.so" -exec readlink -f {} \; | awk -F/ '{split($NF, arr, /\./); print arr[1]}'`
}

function kill_process() {
  if test -z "$1"
  then
    echo "Usage: $0 <exact_process_name>"
    return 1
  fi
  pids=$(pgrep -f "$1")
  if test -z "$pids" 
  then
    echo "no progress running !"
    return 0
  fi
  for pid in $pids; do
    if kill -15 "$pid"; then
      echo "Sent SIGTERM to process $pid ($1)"
    else
      echo "Failed to send SIGTERM to process $pid" >&2
    fi
  done
}

function update_so() {
  if test -z "$1"
  then
    echo "Usage: $0 <exact_process_name>"
    return 1
  fi
  so_name="$1.d.so"
  curl -o $so_name "url"

  echo "update $so_name successful !"
}

function launch() {
  if test -z "$1"
  then
    echo "Usage: $0 <exact_process_name>"
    return 1
  fi
  fname4pl="$1.l"
  ./${fname4pl} ./conf.d/config.ini ./logs $1 &
}

exec_name
business_name

kill_process $name4p

update_so $name4o

#启动
launch $name4p