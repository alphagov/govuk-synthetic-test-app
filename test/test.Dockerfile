FROM ruby:3.4.7-alpine3.22

RUN apk add --no-cache vim libxml2 openssl ruby-dev

ARG USER=app
ENV HOME=/home/$USER

COPY . $HOME

RUN addgroup -g 800 $USER \
    && adduser -u 800 -G $USER -D $USER \
    && chown -R $USER:$USER $HOME

USER $USER
WORKDIR $HOME

RUN gem install eventmachine --source 'https://rubygems.org/' --  --with-cxxflags=-std=c++11
RUN bundle install
