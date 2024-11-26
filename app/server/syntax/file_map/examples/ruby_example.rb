#!/usr/bin/env ruby

global_var = "Hello, World!"

# Module for mixing in common functionality
module Loggable
  def log(message)
    puts "[#{Time.now}] #{message}"
  end
end

# Module with class methods
module Utils
  class << self
    def generate_id
      SecureRandom.uuid
    end
  end
end

# Abstract base class
class BaseProcessor
  include Loggable

  # Class instance variable
  @processors = []

  class << self
    attr_reader :processors

    def register(processor)
      @processors << processor
    end
  end

  # Instance variables with attr accessors
  attr_reader :id, :created_at
  attr_accessor :status

  def initialize
    @id = Utils.generate_id
    @created_at = Time.now
    @status = :pending
    self.class.register(self)
  end

  # Abstract method
  def process
    raise NotImplementedError, "#{self.class} must implement process"
  end
end

# Custom exception class
class ProcessingError < StandardError
  attr_reader :item

  def initialize(message, item)
    @item = item
    super(message)
  end
end

# Struct definition
User = Struct.new(:name, :email, keyword_init: true) do
  def valid?
    name && email && email.include?('@')
  end
end

# Enum-like module using freeze
module Status
  PENDING = 'pending'.freeze
  ACTIVE = 'active'.freeze
  COMPLETED = 'completed'.freeze
  FAILED = 'failed'.freeze

  ALL = [PENDING, ACTIVE, COMPLETED, FAILED].freeze
end

# Class using inheritance and mixins
class DataProcessor < BaseProcessor
  # Constants
  MAX_RETRIES = 3
  DEFAULT_TIMEOUT = 5

  # Class variable
  @@instance_count = 0

  def self.instance_count
    @@instance_count
  end

  def initialize(options = {})
    super()
    @options = options
    @items = []
    @@instance_count += 1
  end

  # Method with keyword arguments and default value
  def add_item(item:, priority: :normal)
    validate_item(item)
    @items << [item, priority]
  end

  # Private methods
  private

  def validate_item(item)
    raise ArgumentError, "Invalid item" unless item.respond_to?(:valid?)
    raise ProcessingError.new("Invalid item", item) unless item.valid?
  end

  # Method using block
  def with_retry
    retries = 0
    begin
      yield
    rescue StandardError => e
      retries += 1
      retry if retries < MAX_RETRIES
      raise
    end
  end

  # Method using lambda
  def process_items
    sorter = ->(a, b) { a[1] <=> b[1] }
    @items.sort(&sorter).each do |item, _priority|
      process_item(item)
    end
  end

  protected

  def process_item(item)
    log("Processing item: #{item}")
    # Processing logic here
  end
end

# Singleton class
require 'singleton'
class Configuration
  include Singleton

  def initialize
    @settings = {}
  end

  def [](key)
    @settings[key]
  end

  def []=(key, value)
    @settings[key] = value
  end
end

# Example usage
if __FILE__ == $PROGRAM_NAME
  config = Configuration.instance
  config[:timeout] = 30

  processor = DataProcessor.new(timeout: config[:timeout])
  user = User.new(name: "John Doe", email: "john@example.com")

  begin
    processor.add_item(item: user, priority: :high)
    processor.process
  rescue ProcessingError => e
    puts "Failed to process #{e.item}: #{e.message}"
  end
end
