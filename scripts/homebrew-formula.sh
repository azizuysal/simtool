#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 || ! "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$ || ! "$2" =~ ^[a-f0-9]{64}$ ]]; then
  echo "Usage: $0 <vMAJOR.MINOR.PATCH> <source archive SHA-256>" >&2
  exit 1
fi

cat <<EOF
class Simtool < Formula
  desc "Terminal UI for iOS Simulator management"
  homepage "https://github.com/azizuysal/simtool"
  url "https://github.com/azizuysal/simtool/releases/download/$1/simtool_${1#v}_source.tar.gz"
  version "${1#v}"
  sha256 "$2"
  license "MIT"

  depends_on "go" => :build
  depends_on macos: :ventura

  def install
    ENV["CGO_ENABLED"] = "1"
    ENV["MACOSX_DEPLOYMENT_TARGET"] = "13.0"
    ldflags = "-s -w -X main.version=v#{version} -X main.builtBy=homebrew"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/simtool"
  end

  test do
    assert_match "simtool version v#{version}", shell_output("#{bin}/simtool --version")
  end
end
EOF
