#!/usr/bin/env ruby
# frozen_string_literal: true

# Unit tests
# To run, simply have Ruby v3.2+ and execute this script
# $ ./handler_test.rb

require 'bundler/inline'
gemfile do
  source 'https://rubygems.org'
  gem 'minitest'
end

require 'json'
require 'minitest/autorun'

# Test the Lambda handler
class LambdaHandlerTest < Minitest::Test
  EVENT_FILE = '../events/event.json'

  def setup
    require_relative '../newrelic_example_ruby/app'
  end

  def test_lambda_handler
    event = JSON.parse(File.read(EVENT_FILE))
    result = App.lambda_handler(event:, context: {})

    assert_kind_of Hash, result, 'Expected a hash result from the Lambda function'
    assert_equal 200, result[:statusCode], 'Expected function result to have a 200 status code'
    assert_match 'Hello', result[:body], "Expected function result message to match 'Hello'"
  end
end
