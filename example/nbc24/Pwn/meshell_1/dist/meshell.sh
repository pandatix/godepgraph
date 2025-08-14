#!/bin/bash

ps1='MeShell $ '

function check() {
    if [[ "$1" =~ [\$,\ ] ]]; then
        return 1
    else
        return 0
    fi
}

while read -p "${ps1}" -r line; do
    if [ "${line}" = "exit" ]; then
        exit
    fi

    if check "$line"; then
        eval "$line"
    else
        echo "nop"
    fi
done
