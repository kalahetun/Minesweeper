#!/bin/bash

# Build script for the Rust Wasm plugin
# This script compiles the Rust code to WebAssembly and optimizes it.

set -e

echo "🦀 Building Rust Wasm plugin..."

# Navigate to the wasm-plugin directory
cd wasm-plugin

# Add the WebAssembly target if not already installed
echo "📦 Adding wasm32-unknown-unknown target..."
rustup target add wasm32-unknown-unknown

# Build the plugin in release mode
echo "🔨 Compiling to WebAssembly..."
cargo build --target wasm32-unknown-unknown --release

# Copy the built .wasm file to the project root for easy access
echo "📋 Copying plugin.wasm to project root..."
cp target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm ../plugin.wasm

echo "✅ Build complete! Plugin available at: $(pwd)/../plugin.wasm"

# Optional: Install wasm-opt and optimize if available
if command -v wasm-opt &> /dev/null; then
    echo "🔧 Optimizing with wasm-opt..."
    wasm-opt -Oz --enable-bulk-memory -o ../plugin.wasm ../plugin.wasm
    echo "✅ Optimization complete!"
else
    echo "💡 Tip: Install 'binaryen' for wasm-opt optimization:"
    echo "   Ubuntu/Debian: sudo apt install binaryen"
    echo "   macOS: brew install binaryen"
fi

echo "🚀 Ready for testing!"
