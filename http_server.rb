require "logger"
require 'rack'
require 'socket'
logger = Logger.new(STDOUT)
server = TCPServer.new 5678
install_id = Time.now.to_i
helm_message = ENV['HELM_MESSAGE'] || 'default_helm_message'

logger.info("From helm chart - #{helm_message}")

Dir['messages/*'].each do |filename|
    file = File.open(filename)
    filedata = file.read
    file.close
    logger.info("#{install_id} - #{filedata}")
end

while session = server.accept
  request = session.gets

  method, full_path = request.split(' ')
  path, query = full_path.split('?')

  logger.info("#{install_id} - #{full_path}")
  
  if path == "/healthcheck/ready"
    session.print "HTTP/1.1 200\r\n"
    session.print "Content-Type: text/html\r\n"
    session.print "\r\n"
    session.print "Hello #{install_id}! The time is #{Time.now}, health check done"
  else
    qs = Rack::Utils.parse_nested_query query
   
    session.print "HTTP/1.1 #{qs["status"] || "400"}\r\n"
    session.print "Content-Type: text/html\r\n"
    session.print "\r\n"
    session.print "Hello #{install_id}! The time is #{Time.now}, you requested a #{qs["status"]} status response"
  end
  
  session.close
end