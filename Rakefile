task :default => [:run]

desc "run server (default port 9000)"
task :run do
  host = ENV['HOST'] || ":9000"
  system %{ HOST=#{host} go run . }
end
