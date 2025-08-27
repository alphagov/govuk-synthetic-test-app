require 'rack'

$install_id = File.read(".version")

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
    req = Rack::Request.new(env)
    path, query = req.fullpath.split('?')

    body = ""
    status = 200

    if path == "/healthcheck/live" || path == "/healthcheck/ready" || path == "/readyz"
      if !req.head?
          body = "Version: #{$install_id}. Hello, the time is #{Time.now}, health check done"
      end
    else 
      if path == "/version"
        $body_message = ""
        body = $install_id
      else
        qs = Rack::Utils.parse_nested_query query
        status = qs["status"] || 400

        puts "path: #{path}"
        if !req.head?
          body = "Version: #{$install_id}. Hello, the time is #{Time.now}, you requested a #{qs["status"]} status response"
        end
      end
    end
    [status, {"Content-Type" => "text/plain"}, [$body_message, body]]
  end
end

run RackApp.new
