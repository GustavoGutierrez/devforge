# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  url "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
  sha256 "3a14fde60c1ecbe04b09ec98f7dca6410d2ba3cc12208103f96d8675ceff2014"

  def install
    libexec.mkpath
    system "curl", "-sSL", "--fail",
           "-o", "#{libexec}/brew-bottle.tar.gz",
           "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
    system "tar", "-xzf", "#{libexec}/brew-bottle.tar.gz",
           "-C", libexec, "--strip-components=1"
    FileUtils.rm_f "#{libexec}/brew-bottle.tar.gz"

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

    bin.install_symlink libexec/"dpf"
  end
end
