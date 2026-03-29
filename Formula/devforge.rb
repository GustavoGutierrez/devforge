# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  version "1.0.1"

  # on_linux overrides these with the real bottle URL below.
  # Homebrew strips the top-level dist/ dir on extraction — binaries land at buildpath root.
  url "https://github.com/GustavoGutierrez/devforge-mcp/archive/refs/tags/v#{version}.tar.gz"
  sha256 "328a7d28132541eebd9bcd9ac0fa5f7a19889c4789a55c435071292be1bba18c"

  on_linux do
    url "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
    sha256 "47eed27d5a44a62bd9c913869419e17c8d24a92b60decff1ac544b0265453026"
  end

  def install
    # Homebrew extracts the bottle, stripping the top-level dist/ directory.
    # The three binaries land directly in buildpath.
    libexec.install "devforge", "devforge-mcp", "dpf"

    # Create bin wrappers
    (libexec/"bin").mkpath
    File.write(libexec/"bin/devforge-mcp", <<~WRAPPER)
      #!/bin/sh
      exec "#{libexec}/devforge-mcp" "$@"
    WRAPPER
    FileUtils.chmod 0755, libexec/"bin/devforge-mcp"

    File.write(libexec/"bin/devforge", <<~WRAPPER)
      #!/bin/sh
      exec "#{libexec}/devforge" "$@"
    WRAPPER
    FileUtils.chmod 0755, libexec/"bin/devforge"

    # Symlink wrappers into bin
    bin.install_symlink libexec/"bin/devforge-mcp" => "devforge-mcp"
    bin.install_symlink libexec/"bin/devforge" => "devforge"
    bin.install_symlink libexec/"dpf"
  end
end
