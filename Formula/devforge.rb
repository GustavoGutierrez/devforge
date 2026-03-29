# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  version "1.1.0"

  url "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
  sha256 "9d646e330cdcaea31ef8633432533c67ba02e2bb9a2b45fb5288e1f30eec2c14"

  def install
    # Create libexec first so we can write into it.
    (libexec).mkpath
    # Download directly into libexec so we control extraction completely.
    # Homebrew's default extraction strips the top-level dist/ dir.
    system "curl", "-sSL", "--fail",
           "-o", "#{libexec}/brew-bottle.tar.gz",
           "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
    system "tar", "-xzf", "#{libexec}/brew-bottle.tar.gz",
           "-C", libexec, "--strip-components=1"
    FileUtils.rm_f "#{libexec}/brew-bottle.tar.gz"

    # Create shell wrappers in bin/
    bin.mkpath
    File.write(bin/"devforge-mcp", <<~WRAPPER)
      #!/bin/sh
      exec "#{libexec}/devforge-mcp" "$@"
    WRAPPER
    FileUtils.chmod 0755, bin/"devforge-mcp"

    File.write(bin/"devforge", <<~WRAPPER)
      #!/bin/sh
      exec "#{libexec}/devforge" "$@"
    WRAPPER
    FileUtils.chmod 0755, bin/"devforge"

    # Symlink dpf so it's in PATH.
    bin.install_symlink libexec/"dpf"
  end
end
