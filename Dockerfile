FROM ruby:3.0.4

WORKDIR /srv/app

COPY ./ /srv/app

RUN bundle install

# CMD ruby http_server.rb
# EXPOSE 5678

EXPOSE 9292

CMD rackup -o 0.0.0.0
