require 'rack'
require 'prometheus_exporter'
require 'prometheus_exporter/server'

server = PrometheusExporter::Server::WebServer.new bind: "0.0.0.0", port: 9394
server.start

$counter = PrometheusExporter::Metric::Counter.new("http_requests_total", "total number of web requests")
$http_request_duration_seconds = PrometheusExporter::Metric::Summary.new("http_request_duration_seconds", "time it took to complete a request", quantiles: [0.01, 0.1, 0.5, 0.9, 0.99])

server.collector.register_metric($counter)
server.collector.register_metric($http_request_duration_seconds)

$install_id = Time.now.to_i

puts("GOVUK synthetic test app - #{$install_id}")
class RackApp
  def call(env)
    start = Time.now.to_f

    req = Rack::Request.new(env)
    path, _ = req.fullpath.split('?')

    body = ""

    if path == "/healthcheck/live" || path == "/healthcheck/ready" || path == "/readyz"
      if !req.head?
          body = "Version: #{$install_id}. Hello, the time is #{Time.now}, health check done"
      end
    else
      body = $install_id
    end

    [200, {"Content-Type" => "text/plain"}, [body]]
  end
end

run RackApp.new
