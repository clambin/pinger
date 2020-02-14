#!/bin/sh

typeset -i I=1

while [ $I -le 10 ]
do
  echo $I
  let I=I+1
done
