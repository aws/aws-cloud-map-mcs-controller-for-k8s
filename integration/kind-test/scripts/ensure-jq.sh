#!/usr/bin/env bash

# Ensure jq is available to parse json output. Installs jq on debian/ubuntu

if ! which -s jq ; then
  echo "jq not found, attempting to install"
  if ! sudo apt-get install -y jq ; then
    echo "failed to install jq, ensure it is available before running tests"
    exit 1
  fi
fi

exit 0
