#!/usr/bin/env bash

nohup ./ahri-client \
-sip yourIP \
-sp yourPort \
-k yourPswd \
-n yourClientName \
-m 0 \
-s5ip 127.0.0.1 \
-s5p 23456 \
-f ahri.hosts \
-L 3 \
-T 5 \
>./a.log 2>&1 &

exit 0
