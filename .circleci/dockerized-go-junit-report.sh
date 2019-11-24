#!/bin/bash

# see if aws is available, if not fallback to local go-junit-report
aws --version >/dev/null 2>&1
if [ $? == 0 ]; then
  $(aws ecr get-login --no-include-email --region us-west-2) >/dev/null 2>&1
  docker run -i 727534866935.dkr.ecr.us-west-2.amazonaws.com/go-junit-report:latest $@
else
  go-junit-report $@
fi

