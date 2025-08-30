#!/bin/bash
RESPUESTA=$(docker run --rm --network=tp0_testing_net alpine /bin/sh -c 'echo "mensaje" | nc server 12345')

if [ "$RESPUESTA" == "mensaje" ]; then
  echo "action: test_echo_server | result: success"
else
  echo "action: test_echo_server | result: fail"
fi