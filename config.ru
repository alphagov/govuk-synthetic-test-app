require 'rack'
require 'prometheus_exporter'
require 'prometheus_exporter/server'

server = PrometheusExporter::Server::WebServer.new bind: "0.0.0.0", port: 9394
server.start

$counter = PrometheusExporter::Metric::Counter.new("http_requests_total", "total number of web requests")
server.collector.register_metric($counter)

$install_id = Time.now.to_i
helm_message = ENV['HELM_MESSAGE'] || 'missing_helm_message'

puts("GOVUK replatform test app - #{$install_id} - from helm chart - #{helm_message}")

Dir['messages/*'].each do |filename|
    file = File.open(filename)
    filedata = file.read
    file.close
    puts("#{$install_id} - #{filedata}")
end

class RackApp    
  def call(env)
    req = Rack::Request.new(env)
    path, query = req.fullpath.split('?')

    body = ""
    status = 200

    if path == "/healthcheck/live" || path == "/healthcheck/ready" || path == "/readyz"
      if !req.head?
          body = "Hello #{$install_id}! The time is #{Time.now}, health check done"
      end
    else
      $counter.observe(1, route: '/')

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
