require 'rack'
require 'prometheus_exporter'
require 'prometheus_exporter/server'

server = PrometheusExporter::Server::WebServer.new bind: "0.0.0.0", port: 9394
server.start

$counter = PrometheusExporter::Metric::Counter.new("http_requests_total", "total number of web requests")
$http_request_duration_seconds = PrometheusExporter::Metric::Summary.new("http_request_duration_seconds", "time it took to complete a request", quantiles: [0.01, 0.1, 0.5, 0.9, 0.99])

server.collector.register_metric($counter)
server.collector.register_metric($http_request_duration_seconds)

version_file = File.open(".version")
$install_id = version_file.readline
version_file.close

$body_message = "Start ENV_MESSAGEs:\n"

(ENV.keys.select { |k| k.start_with?("ENV_MESSAGE") }).each do |k|
  $body_message += "#{$install_id} - #{ENV[k]}\n"
end

$body_message += "End ENV_MESSAGEs.\n"

puts("GOVUK replatform test app - #{$install_id}\n#{$body_message}")

Dir['messages/*'].each do |filename|
    file = File.open(filename)
    filedata = file.read
    file.close
    $body_message += "#{filedata}\n"
    puts("#{$install_id} - #{filedata}")
end

class RackApp
  def call(env)
    start = Time.now.to_f

    req = Rack::Request.new(env)
    path, query = req.fullpath.split('?')

    body = ""
    status = 200

    if path == "/healthcheck/live" || path == "/healthcheck/ready" || path == "/readyz"
      if !req.head?
          body = "Version: #{$install_id}. Hello, the time is #{Time.now}, health check done"
      end
    else
      qs = Rack::Utils.parse_nested_query query
      status = qs["status"] || 400

      if !req.head?
        body = "Version: #{$install_id}. Hello, the time is #{Time.now}, you requested a #{qs["status"]} status response"
      end

      # $counter.observe(1, route: path, status: status, install_id: $install_id)

      # duration = Time.now.to_f - start
      # $http_request_duration_seconds.observe(duration, action: 'test', status: status, install_id: $install_id)
    end
    [status, {"Content-Type" => "text/plain"}, [$body_message, body]]
  end
end

run RackApp.new
