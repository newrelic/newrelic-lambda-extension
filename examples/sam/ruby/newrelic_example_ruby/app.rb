# frozen_string_literal: true

require 'net/http'
require 'uri'

# Example Ruby Lambda function - module and/or class
# namespacing is optional
class App
  def self.lambda_handler(event:, context:)
    # instrumentation
    uri = URI('https://newrelic.com')
    3.times { Net::HTTP.get(uri) }

    # custom attributes
    # ::NewRelic::Agent.add_custom_attributes(server: 'less', current_time: Time.now.to_s)

    # As normal, anything you write to stdout ends up in CloudWatch
    puts 'Hello, world'
    puts "Event size: #{event.size}"
    { statusCode: 200, body: JSON.generate('Hello from Ruby Lambda!') }
  end
end
