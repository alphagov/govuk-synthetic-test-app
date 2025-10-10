FROM ruby:3.2.9-alpine3.22

RUN apk add --no-cache vim libxml2 openssl ruby-dev

ARG USER=app
ENV HOME=/home/$USER

COPY . $HOME

RUN addgroup -g 800 $USER \
    && adduser -u 800 -G $USER -D $USER \
    && chown -R $USER:$USER $HOME

USER $USER
WORKDIR $HOME

RUN bundle install
