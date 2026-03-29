# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  url "https://github.com/GustavoGutierrez/devforge-mcp/archive/refs/tags/v#{version}.tar.gz"
  sha256 "328a7d28132541eebd9bcd9ac0fa5f7a19889c4789a55c435071292be1bba18c"

  def install
    # Download and extract the pre-built Linux bottle.
    # We do this manually because the source archive does not contain dist/.
    bottle_url = "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v1.0.1/devforge-1.0.1.linux-amd64.tar.gz"
    system "curl", "-sSL", "--fail", "-o", "brew-bottle.tar.gz", bottle_url
    system "tar", "-xzf", "brew-bottle.tar.gz"
    FileUtils.rm_f "brew-bottle.tar.gz"

    # The tarball extracts into dist/
    libexec.install "dist/devforge-mcp", "dist/devforge", "dist/dpf"

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
  end
end
