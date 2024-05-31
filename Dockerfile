##############################################################
## Stage 1 - Go Build
##############################################################
FROM golang:1.21.6-alpine AS builder

RUN apk add --no-cache build-base


RUN mkdir -p /src/mc-iam-manager
WORKDIR /src/mc-iam-manager
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

RUN wget https://github.com/gobuffalo/cli/releases/download/v0.18.14/buffalo_0.18.14_Linux_x86_64.tar.gz
RUN tar -xvzf buffalo_0.18.14_Linux_x86_64.tar.gz
RUN mv buffalo /usr/local/bin/buffalo

ADD . .
RUN buffalo build --static -o /bin/app

FROM alpine
RUN apk add --no-cache bash
RUN apk add --no-cache ca-certificates

WORKDIR /bin/
ADD conf /bin/conf

ENV CBSPIDER_ROOT=/bin \
    CBSTORE_ROOT=/bin \
    CBLOG_ROOT=/bin

COPY --from=builder /bin/app .

# Uncomment to run the binary in "production" mode:
# ENV GO_ENV=production
ENV GO_ENV=development

# Bind the app to 0.0.0.0 so it can be seen from outside the container
ENV ADDR=0.0.0.0
EXPOSE 3000

# Uncomment to run the migrations before running the binary:
# CMD /bin/app migrate; /bin/app
CMD exec /bin/app