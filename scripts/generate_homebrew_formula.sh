#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 5 ]; then
  echo "usage: $0 <version> <darwin_amd64_sha> <darwin_arm64_sha> <linux_amd64_sha> <linux_arm64_sha>" >&2
  exit 1
fi

version="$1"
darwin_amd64_sha="$2"
darwin_arm64_sha="$3"
linux_amd64_sha="$4"
linux_arm64_sha="$5"

cat <<EOF
class Lazytask < Formula
  desc "Terminal-first task manager inspired by LazyGit/LazyVim"
  homepage "https://github.com/Joseda-hg/lazytask"
  version "${version}"

  on_macos do
    on_arm do
      url "https://github.com/Joseda-hg/lazytask/releases/download/v${version}/lazytask_${version}_darwin_arm64.tar.gz"
      sha256 "${darwin_arm64_sha}"
    end
    on_intel do
      url "https://github.com/Joseda-hg/lazytask/releases/download/v${version}/lazytask_${version}_darwin_amd64.tar.gz"
      sha256 "${darwin_amd64_sha}"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Joseda-hg/lazytask/releases/download/v${version}/lazytask_${version}_linux_arm64.tar.gz"
      sha256 "${linux_arm64_sha}"
    end
    on_intel do
      url "https://github.com/Joseda-hg/lazytask/releases/download/v${version}/lazytask_${version}_linux_amd64.tar.gz"
      sha256 "${linux_amd64_sha}"
    end
  end

  def install
    bin.install "lazytask"
  end
end
EOF
