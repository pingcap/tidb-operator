FROM alpine:latest
ARG TARGETARCH
RUN apk add lsblk fio
ADD bin/${TARGETARCH}/warmup /warmup

ADD warmup-steps.sh /warmup_steps
CMD /warmup_steps
