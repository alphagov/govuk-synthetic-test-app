FROM golang:1.27.3-alpine3.24

ARG USER=app
ENV HOME=/home/$USER

COPY . $HOME

RUN addgroup -g 1000 $USER \
    && adduser -u 1000 -G $USER -D $USER \
    && chown -R $USER:$USER $HOME

RUN apk add --no-cache git

USER $USER
WORKDIR $HOME

RUN go mod vendor
RUN go install github.com/onsi/ginkgo/v2/ginkgo

CMD ["ginkgo", "-v", "helpers"]
