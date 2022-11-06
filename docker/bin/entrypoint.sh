#!/bin/sh
set -e

# started as hassio addon
HASSIO_OPTIONSFILE=/data/options.json

if [ -f ${HASSIO_OPTIONSFILE} ]; then
    API=$(grep -o '"api": "[^"]*' ${HASSIO_OPTIONSFILE} | grep -o '[^"]*$')
    echo "Using api endpoint: ${API}"

    if [ ! -f "${API}" ]; then
        echo "api not found. Please set up this option in the addon configuration."
    else
        gravo -api "${API}"
    fi
else
    gravo
fi
