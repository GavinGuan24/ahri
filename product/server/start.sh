#!/usr/bin/env bash

bash gen_rsa_keys.sh

nohup ./ahri-server \
-ip yourIP \
-p yourPort \
-k yourPswd \
-a rsa_private_key.pem \
-b rsa_public_key.pem \
-L 3 \
-T 5 \
>./a.log 2>&1 &

exit 0
