require 'octokit'
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
      if path.start_with?("/version")
        $body_message = ""
        if path == "/version/increment"
          $install_id = (Integer($install_id) + 1).to_s
          body = $install_id
          
          branch="update-version"

          %x(git checkout -b "#{branch}")
          %x(git pull origin "#{branch}")
          %x(git checkout -b "update-version")

          File.write(".version", $install_id)
          %x(git add ".version")
          %x(git commit -m "Update version to to #{$install_id}")
          %x(git push --set-upstream origin "#{$install_id}")
          # octokit = Octokit::Client.new(access_token: '')

        else
          body = $install_id
        end
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
