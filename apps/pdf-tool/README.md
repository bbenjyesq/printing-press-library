# PDF Tool

A small Mac app (Tauri) that does two things to a PDF: make it smaller, or add a password.
Everything runs on your Mac; no file leaves the machine. The original is never changed —
the app always saves a copy.

## Get the app

Every push that touches this folder builds a `.dmg` for Apple-silicon Macs: GitHub →
Actions → "Build PDF Tool (macOS)" → latest run → Artifacts → `PDF-Tool-macOS`.

The build is not signed with an Apple Developer ID, so the first launch is blocked. Open
it once, then go to System Settings → Privacy & Security → "Open Anyway".

## Build it yourself (on the Mac)

```sh
xcode-select --install                                           # once
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh   # once
cd apps/pdf-tool
npx @tauri-apps/cli@2 build        # needs Node; or: cargo install tauri-cli && cargo tauri build
open src-tauri/target/release/bundle/dmg
```

## How it works

- `core/` — the PDF engine (Rust, `lopdf` + `image`). `cargo test` in that folder runs it
  against a generated scan. Reduce: images are capped at 200/150/100 dpi-equivalent for a
  letter page and re-saved as JPEG; text, fonts and vector art are untouched. Password:
  AES-256, required to open the file.
- `src-tauri/` — the window and the open/save dialogs.
- `ui/index.html` — the whole interface; no build step.
