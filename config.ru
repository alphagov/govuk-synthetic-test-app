require "logger"
require 'rack'
require 'prometheus/middleware/collector'
require 'prometheus/middleware/exporter'

use Rack::Deflater
use Prometheus::Middleware::Collector
use Prometheus::Middleware::Exporter

$install_id = Time.now.to_i

puts "#{$install_id} - rackup"

class RackApp    
  def call(env)
    req = Rack::Request.new(env)
    path, query = req.fullpath.split('?')

    body = ""
    status = 200

    if path == "/healthcheck/ready" || path == "/readyz"
      if !req.head?
          body = "Hello #{$install_id}! The time is #{Time.now}, health check done"
      end
    else
      qs = Rack::Utils.parse_nested_query query
      status = qs["status"] || 400
      if !req.head?
        body = "Hello #{$install_id}! The time is #{Time.now}, you requested a #{qs["status"]} status response"
      end
    end
    [status, {"Content-Type" => "text/plain"}, [body]]
end
end

run RackApp.new
