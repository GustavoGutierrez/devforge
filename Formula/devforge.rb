# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  url "https://github.com/GustavoGutierrez/devforge-mcp/archive/refs/tags/v#{version}.tar.gz"
  sha256 "328a7d28132541eebd9bcd9ac0fa5f7a19889c4789a55c435071292be1bba18c"

  if OS.linux?
    on_linux do
      url "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v1.0.1/devforge-1.0.1.linux-amd64.tar.gz"
      sha256 "47eed27d5a44a62bd9c913869419e17c8d24a92b60decff1ac544b0265453026"
    end
  end

  def install
    libexec.install "dist/devforge-mcp", "dist/devforge", "dist/dpf"

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
  end
<<<<<<< HEAD
=======

  def install
    libexec.install "dist/devforge-mcp", "dist/devforge", "dist/dpf"

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
>>>>>>> e802b04 (fix(homebrew): use explicit file paths instead of Dir glob in install method)
end
