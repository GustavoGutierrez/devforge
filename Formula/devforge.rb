# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge — Design intelligence (CLI/TUI + MCP server) + image/video/audio processing (dpf)"
  homepage "https://github.com/GustavoGutierrez/devforge-mcp"
  license "GPL-3.0"

  url "https://github.com/GustavoGutierrez/devforge-mcp/archive/refs/tags/v#{version}.tar.gz"
  version "#{version}"
  sha256 "328a7d28132541eebd9bcd9ac0fa5f7a19889c4789a55c435071292be1bba18c"

  if OS.linux?
    on_linux do
      url "https://github.com/GustavoGutierrez/devforge-mcp/releases/download/v#{version}/devforge-#{version}.linux-amd64.tar.gz"
      sha256 "da92aa49eb359fc6e33af38836a6e8bfb2498be4751b017b49f6bdf8480cd22e"
    end
  end
