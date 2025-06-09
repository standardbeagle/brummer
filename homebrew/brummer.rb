class Brummer < Formula
  desc "🐝 Terminal UI for managing npm/yarn/pnpm/bun scripts with intelligent monitoring"
  homepage "https://github.com/beagle/brummer"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/beagle/brummer/releases/download/v#{version}/brum-darwin-amd64"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"
    elsif Hardware::CPU.arm?
      url "https://github.com/beagle/brummer/releases/download/v#{version}/brum-darwin-arm64"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/beagle/brummer/releases/download/v#{version}/brum-linux-amd64"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    elsif Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/beagle/brummer/releases/download/v#{version}/brum-linux-arm64"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    end
  end

  def install
    bin.install Dir["brum-*"].first => "brum"
  end

  test do
    # Test that the binary runs
    assert_match "Brummer", shell_output("#{bin}/brum --version")
  end
end