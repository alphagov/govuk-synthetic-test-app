require 'rack'

$install_id = File.read(".version")

puts("GOVUK synthetic test app - #{$install_id}")
class RackApp
  def call(env)
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
