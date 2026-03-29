# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  version "1.1.1"

  url "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
  sha256 "157a63e99732d0c524f77320e4e3068b0d2f8f2ae0b13b3dfe84419e24fdc3b8"

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
