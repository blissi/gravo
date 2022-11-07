#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
    API=$(grep -o '"api": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')
    echo "Using api endpoint: ${API}"
    gravo -api "${API}"
else
    gravo
fi
