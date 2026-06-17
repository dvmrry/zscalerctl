class Zscalerctl < Formula
  desc "Read-only CLI for Zscaler configuration query, inventory, and sanitized exports"
  homepage "https://github.com/dvmrry/zscalerctl"
  version "0.60.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/dvmrry/zscalerctl/releases/download/v#{version}/zscalerctl_#{version}_darwin_arm64.tar.gz"
      sha256 "85eb81b753a369ef79348031e0245354502287bb544b6d8fb745592c0d04c901"
    end
    on_intel do
      url "https://github.com/dvmrry/zscalerctl/releases/download/v#{version}/zscalerctl_#{version}_darwin_amd64.tar.gz"
      sha256 "7ea6f2c91c599fa46d1615a2f1d540eca485db583e408afb3a0b278446cd1cd4"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/dvmrry/zscalerctl/releases/download/v#{version}/zscalerctl_#{version}_linux_arm64.tar.gz"
      sha256 "42b53f667cbabec1a1360568a4f523adab0048e9fd2b201c257965f576518f47"
    end
    on_intel do
      url "https://github.com/dvmrry/zscalerctl/releases/download/v#{version}/zscalerctl_#{version}_linux_amd64.tar.gz"
      sha256 "779889bab69c62236eb16ecd5ea385fdfb80f4ecb55dbb71816cf37d7c2d2543"
    end
  end

  # Load-bearing: `brew livecheck` and the auto-bump action use this to detect
  # new upstream releases.
  livecheck do
    url :url
    strategy :github_latest
  end

  def install
    bin.install "zscalerctl"
    man1.install "man/zscalerctl.1"
    generate_completions_from_executable(bin/"zscalerctl", "completion")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zscalerctl version")
  end
end
