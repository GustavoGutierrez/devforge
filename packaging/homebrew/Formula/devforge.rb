# frozen_string_literal: true

class Devforge < Formula
  desc "DevForge CLI, MCP server, and bundled DevPixelForge runtime"
  homepage "https://github.com/GustavoGutierrez/devforge"
  license "GPL-3.0"
  version "__VERSION__"

  on_linux do
    depends_on arch: :x86_64

    url "https://github.com/GustavoGutierrez/devforge/releases/download/v#{version}/devforge_#{version}_linux_amd64.tar.gz"
    sha256 "__LINUX_AMD64_SHA256__"
  end

  on_macos do
    depends_on arch: :arm64

    url "https://github.com/GustavoGutierrez/devforge/releases/download/v#{version}/devforge_#{version}_darwin_arm64.tar.gz"
    sha256 "__DARWIN_ARM64_SHA256__"
  end

  def install
    # dpf is bundled as a pre-built binary from DevPixelForge releases.
    # The bundle build step downloads dpf and renames it to "dpf" before packaging.
    # DevPixelForge release archives use platform-specific binary names:
    #   dpf-linux-amd64.tar.gz  → binary inside is: dpf-dpf-linux-amd64
    #   dpf-macos-arm64.tar.gz  → binary inside is: dpf-dpf-macos-arm64
    # We handle all known names defensively in case the bundle ever ships the original name.
    dpf_src = %w[dpf dpf-dpf-linux-amd64 dpf-dpf-macos-arm64].find { |f| File.exist?(f) }
    raise "dpf binary not found in bundle (expected: dpf, dpf-dpf-linux-amd64, or dpf-dpf-macos-arm64)" unless dpf_src

    bin.install "devforge", "devforge-mcp"
    bin.install dpf_src => "dpf"
  end

  def post_install
    config_dir = Pathname.new(Dir.home)/".config"/"devforge"
    config_file = config_dir/"config.json"
    return if config_file.exist?

    config_dir.mkpath
    config_file.write <<~JSON
      {
        "gemini_api_key": "",
        "image_model": "gemini-2.5-flash-image"
      }
    JSON
    config_file.chmod 0o600
  end

  def caveats
    <<~EOS
      DevForge installs a utility-focused runtime bundle with:

        - devforge       (CLI/TUI)
        - devforge-mcp   (MCP stdio server)
        - dpf            (DevPixelForge media engine)

      Config file: ~/.config/devforge/config.json

      For media processing tools (video/audio via dpf), install ffmpeg:
        brew install ffmpeg
    EOS
  end

  test do
    assert_predicate bin/"devforge", :executable?
    assert_predicate bin/"devforge-mcp", :executable?
    assert_predicate bin/"dpf", :executable?

    initialize_request = <<~JSON
      {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"brew-test","version":"0.1"}}}
    JSON

    output = pipe_output("#{bin}/devforge-mcp", initialize_request, 0)
    assert_match("serverInfo", output)
  end
end
