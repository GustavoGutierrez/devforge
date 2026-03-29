# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  version "1.0.1"

  url "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
  sha256 "47eed27d5a44a62bd9c913869419e17c8d24a92b60decff1ac544b0265453026"

  def install
    # Download the Linux bottle directly from GitHub Releases.
    # Homebrew strips the top-level dist/ dir on extraction, so we
    # bypass its extraction by downloading and extracting manually.
    system "curl", "-sSL", "--fail",
           "-o", "brew-bottle.tar.gz",
           "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
    system "tar", "-xzf", "brew-bottle.tar.gz"
    FileUtils.rm_f "brew-bottle.tar.gz"
    libexec.install "dist/devforge", "dist/devforge-mcp", "dist/dpf"

    # Create shell wrappers in bin/
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
  end
end
