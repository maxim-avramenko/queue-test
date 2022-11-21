#!/usr/bin/env bash

# Необходимо скопировать sudo cp queue.conf /etc/supervisor/conf.d/queue.conf
# и применить sudo supervisorctl reread

while true
do
  # get total messages in queue from RabbitMq
  total=$(/usr/local/bin/docker-compose exec queue-rabbitmq bash -c "rabbitmqctl list_queues | grep urls | awk '{print $2}'")

  if (( total > 10 )); then
  # Запустим worker в 5 потоков
    /usr/local/bin/docker-compose up -d queue-worker --scale queue-worker=5
    echo `date` "5 worker online"
  else
  # оставим один worker как дежурный
    /usr/local/bin/docker-compose up -d queue-worker --scale queue-worker=1
    echo `date` "1 worker online"
  fi;
  sleep 10
done