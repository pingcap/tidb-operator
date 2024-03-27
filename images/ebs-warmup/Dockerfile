FROM pingcap/tikv:v6.5.7
ARG TARGETARCH
RUN apk add lsblk fio bash
ADD bin/${TARGETARCH}/warmup /warmup

ADD warmup-steps.sh /warmup_steps
CMD /warmup_steps
ENTRYPOINT /bin/sh
