#!/usr/bin/env bash

# This is a workaround script that will try to map any variable 
# ending in "__FILE" into a Docker secret and load it into the environment.
# If the environment variable is already in use, no action is taken.

# build list of candidate environment variables
SECRETVARS=`compgen -v | grep '__FILE$'`

# iterate over list and parse the base variable name out from the 
# "__FILE" version of the name. 
# if:
#  a) basename doesn't exist
#  b) is not empty 
#  c) points to a readable file
# then:
#  create a new variable using the base name and file contents
while read VARNAME; do
   BASE_VARNAME=`echo ${VARNAME} | awk -F '__' '{print $1}';`
   if ! [ -v "${BASE_VARNAME}" ] && [ -n "${VARNAME}" ] && [ -r "${!VARNAME}" ]; then       
      declare -x "${BASE_VARNAME}=$(<${!VARNAME})"
   fi;
done <<< "${SECRETVARS}"

# pass along any args as a single string
exec /app/pihole-exporter $*

