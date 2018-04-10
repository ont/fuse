#
# NOTE: this Dockerfile is only for packaging, actual build steps are in .drone.yml
#
FROM alpine

ADD ./bin/fuse /fuse

# SEE: https://stackoverflow.com/a/35613430/434255
# fix error "/bin/sh: /server: not found" 
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

# fix error "x509: failed to load system roots and no roots provided"
RUN apk add --no-cache ca-certificates

EXPOSE 7777 7778

ENTRYPOINT ["/fuse"]
