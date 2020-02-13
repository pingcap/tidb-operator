FROM bash:4.3.48
RUN wget -q http://download.pingcap.org/tidb-latest-linux-amd64.tar.gz \
        && tar xzf tidb-latest-linux-amd64.tar.gz \
        && mv tidb-*-linux-amd64/bin/pd-ctl \
              tidb-*-linux-amd64/bin/tidb-ctl \
              /usr/local/bin/ \
        && rm -rf tidb-latest-linux-amd64.tar.gz tidb-*-linux-amd64

ADD banner /etc/banner
ADD profile  /etc/profile

CMD ["/usr/local/bin/bash", "-l"]
