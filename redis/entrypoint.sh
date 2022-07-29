#!/bin/sh

set -e

ME=$(basename $0)

run_envsubst() {
    local template defined_envs output_path
    template="$1"
    output_path="$2"
    defined_envs=$(printf '${%s} ' $(env | cut -d= -f1))
    echo "$ME: Running envsubst on $template to $output_path"
    envsubst "$defined_envs" < "$template" > "$output_path"
}

if [ -f "/templates/redis.conf.template" ]; then
    mkdir -p /usr/local/etc/redis
    run_envsubst "/templates/redis.conf.template" "/usr/local/etc/redis/redis.conf"
fi

redis-server /usr/local/etc/redis/redis.conf