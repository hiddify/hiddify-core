FROM alpine:latest
ENV CONFIG='https://raw.githubusercontent.com/ircfspace/warpsub/main/export/warp#WARP%20(IRCF)'
ENV VERSION=v0.17.8
WORKDIR /opt
RUN apk add  wget tar gzip libc6-compat
RUN export ARCH=$(apk --print-arch) && if [ "$ARCH" = "i386" ]; then ARCH="386"; elif [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi && wget https://github.com/hiddify/hiddify-next-core/releases/download/${VERSION}/hiddify-cli-linux-$ARCH.tar.gz -O hiddify-cli.tar.gz
RUN tar -xzf hiddify-cli.tar.gz && rm hiddify-cli.tar.gz
COPY hiddify.sh .
COPY hiddify.json .
RUN chmod +x hiddify.sh

EXPOSE 2334
EXPOSE 6756
EXPOSE 6450


ENTRYPOINT [ "/opt/hiddify.sh" ]