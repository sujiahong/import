#!/bin/sh

filename="$1"
filtercondtion="$2"
step="$3"

function first_step() {
  echo "first step", $filename, $filtercondtion
  find ./?/ -name ${filename} -mmin -100 | xargs grep " Recv" | grep ${filtercondtion}
}

condition="$4"
function second_step() {
  echo "second step", $filename, $filtercondtion, ${condition}
  find ./?/ -name ${filename} | xargs grep ${filtercondtion} | grep $condition
}

function third_step() {
  echo "third step", $filename, $filtercondtion
  find ./?/ -name ${filename} | xargs grep ${filtercondtion}
}

if [ ${step} == "1" ]
then
  first_step
elif [ ${step} == "2" ]
then
  second_step
elif [ ${step} == "3" ]
then
  third_step
fi