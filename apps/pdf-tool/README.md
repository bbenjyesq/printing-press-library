# PDF Tool

A small Mac app (Tauri) for PDFs: make them smaller, add a password (or both in one
pass), or remove a password you know. Drop one file, several, or a folder. Everything runs on your Mac; no
file leaves the machine.

Originals are never changed. Each copy is saved next to its original as `name-small.pdf`,
`name-protected.pdf`, `name-small-protected.pdf` or `name-unlocked.pdf`; if that name is taken, ` 2`, ` 3`… is
added rather than overwriting. "Show in Finder" jumps to the result.

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
- `src-tauri/` — the window, the file picker, and saving the copies (`cargo test` there
  checks the never-overwrite naming).
- `ui/index.html` — the whole interface; no build step.
