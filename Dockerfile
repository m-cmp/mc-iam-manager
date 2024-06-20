##############################################################
## Stage 1 - Go Build
##############################################################

FROM golang:1.22.3-alpine AS builder

RUN apk add wget
RUN apk add --no-cache sqlite-libs sqlite-dev build-base
RUN mkdir -p /util
WORKDIR /util
RUN wget https://github.com/gobuffalo/cli/releases/download/v0.18.14/buffalo_0.18.14_Linux_x86_64.tar.gz \
    && tar -xvzf buffalo_0.18.14_Linux_x86_64.tar.gz \
    && mv buffalo /usr/local/bin/buffalo \
    && rm buffalo_0.18.14_Linux_x86_64.tar.gz

RUN mkdir -p /src/mc-iam-manager
WORKDIR /src/mc-iam-manager
ENV GOPROXY http://proxy.golang.org
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

ADD . .
RUN buffalo build --static -o /bin/app

#############################################################
## Stage 2 - Application Deploy
##############################################################

FROM debian:buster-slim
WORKDIR /bin/
COPY --from=builder /bin/app .
# ENV GO_ENV=production
ENV ADDR=0.0.0.0 \
    PORT=3000
EXPOSE 3000
CMD bash -c 'until /bin/app migrate; do echo "Migration failed. Retrying in 10 seconds..."; sleep 10; done; /bin/app'
