task :default => [:run]

desc "run server (default port 9002)"
task :run do
  host = ENV['HOST'] || ":9002"
  secret = ENV['HMAC_SECRET']
  header = ENV['HMAC_HEADER']

  cmd_args = ["-listen", host]
  cmd_args << "-hmac-secret" << secret if secret
  cmd_args << "-hmac-header-name" << header if header
  
  puts "#{cmd_args}"
  
  system %{ go run . #{cmd_args.join(" ")} }
end


task :command_exists, [:command] do |_, args|
  abort "#{args.command} doesn't exists" if `command -v #{args.command} > /dev/null 2>&1 && echo $?`.chomp.empty?
end
task :is_repo_clean do
  abort 'please commit your changes first!' unless `git status -s | wc -l`.strip.to_i.zero?
end
task :has_bumpversion do
  Rake::Task['command_exists'].invoke('bumpversion')
end


AVAILABLE_REVISIONS = %w[major minor patch].freeze
task :bump, [:revision] => [:has_bumpversion] do |_, args|
  args.with_defaults(revision: 'patch')
  unless AVAILABLE_REVISIONS.include?(args.revision)
    abort "Please provide valid revision: #{AVAILABLE_REVISIONS.join(',')}"
  end

  system "bumpversion #{args.revision}"
  exit $?.exitstatus
end

desc "release new version #{AVAILABLE_REVISIONS.join(',')}, default: patch"
task :release, [:revision] => [:is_repo_clean] do |_, args|
  args.with_defaults(revision: 'patch')
  Rake::Task['bump'].invoke(args.revision)
end

DOCKER_IMAGE_NAME = "basichttpdebugger:latest"

namespace :docker do
  desc "build docker image locally"
  task :build do
    system %{
      docker build -t #{DOCKER_IMAGE_NAME} .
    }
  end

  desc "run docker image locally"
  task :run do
    system %{
      docker run -p "9002:9002" #{DOCKER_IMAGE_NAME}
    }
  end
end