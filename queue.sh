#!/usr/bin/env bash

#set -eux
set -e

WORKERS_MAX_LIMIT="${1:-20}"
WORKERS_MIN_LIMIT="${2:-1}"
LIMIT=${WORKERS_MIN_LIMIT}
read -ra WHEREIS_DOCKER_COMPOSE < <(whereis docker-compose)
DC="${WHEREIS_DOCKER_COMPOSE[1]}"

# check docker-compose on host
if [[ -z "${DC:-}" ]]; then
  1>&2 echo `date` " | [ERROR] queue-worker: Please install docker-compose."
  exit 1
fi;

countMessages()
{
  local RESPONSE
  local TOTAL_URLS
  RESPONSE="$("${DC}" exec queue-rabbitmq bash -c 'rabbitmqctl list_queues -p queue_rabbitmq')"
  TOTAL_URLS="$(echo "${RESPONSE}" | grep urls | awk '{print $2}')"
  echo "${TOTAL_URLS}"
}

countWorkers()
{
  local WORKER_LIST
  WORKER_LIST="$("${DC}" ps | grep queue-worker | awk '{print $1}' | grep -c '^')"
  echo "${WORKER_LIST}"
}

updateQueueWorkers()
{
  "${DC}" -f docker-compose.prod.yml up -d queue-worker --scale queue-worker="${LIMIT}" > /dev/null
  echo `date`" | [INFO] Started ${LIMIT} worker."
}

while true
do
  TOTAL_MESSAGES="$(countMessages)"
  TOTAL_WORKERS="$(countWorkers)"
  if [[ "${TOTAL_WORKERS}" -eq "${WORKERS_MAX_LIMIT}" ]] && (( TOTAL_MESSAGES > 0 )); then
      sleep 10
      continue
  fi;
  if [[ "${TOTAL_WORKERS}" -eq "${WORKERS_MIN_LIMIT}" ]] && (( TOTAL_MESSAGES == 0 )); then
      sleep 10
      continue
  else
      if (( TOTAL_MESSAGES > 0 )) && (( TOTAL_WORKERS == WORKERS_MIN_LIMIT )); then
        LIMIT=${WORKERS_MAX_LIMIT}
      fi;
      if (( TOTAL_MESSAGES == 0 )) && (( TOTAL_WORKERS > WORKERS_MIN_LIMIT )) ; then
        LIMIT=${WORKERS_MIN_LIMIT}
      fi;
      updateQueueWorkers
  fi;
  sleep 10
done