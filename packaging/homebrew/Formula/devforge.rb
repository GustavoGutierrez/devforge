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

  # Future work: publish a darwin/arm64 bundle and add an `on_macos` block.
  depends_on "ffmpeg"

  def install
    libexec.install "devforge", "devforge-mcp", "dpf", "devforge.db"

    (bin/"devforge").write_env_script libexec/"devforge"
    (bin/"devforge-mcp").write_env_script libexec/"devforge-mcp"
    bin.install_symlink libexec/"dpf"
  end

  def post_install
    config_dir = Dir.home.join(".config", "devforge")
    config_file = config_dir.join("config.json")
    return if config_file.exist?

    config_dir.mkpath
    config_file.write <<~JSON
      {
        "gemini_api_key": "",
        "ollama_url": "http://localhost:11434",
        "embedding_model": "nomic-embed-text",
        "image_model": "gemini-2.5-flash-image"
      }
    JSON
    config_file.chmod 0o600
  end

  def caveats
    <<~EOS
      DevForge installs a Linux runtime bundle with:

        - devforge       (CLI/TUI)
        - devforge-mcp   (MCP stdio server)
        - dpf            (DevPixelForge media engine)
        - devforge.db    (seeded runtime database)

      Config file: ~/.config/devforge/config.json

      The Homebrew formula is Linux amd64 first.
      macOS arm64 support is planned in a future release.
    EOS
  end

  test do
    assert_predicate libexec/"devforge.db", :exist?
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
