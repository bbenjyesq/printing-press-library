# PDF Tool — handoff

State as of 2026-11-24. Written for whoever (person or agent) picks this up next.

## What it is

A small macOS app (Tauri v2, Apple silicon) for an office Mac. One window, three tabs:

- **Reduce size** — Light / Balanced / Smallest, with an optional "Also add a password".
- **Add password** — AES-256, required to open the file.
- **Remove password** — for a password the user knows.

It accepts one file, several, or a dropped folder (PDFs directly inside it, not subfolders).
Everything runs locally. Originals are never modified.

## Where things are

- Repository: `bbenjyesq/printing-press-library` (an unrelated CLI catalog; the app just lives here).
- Branch: `claude/pdf-utility-mac-app-x53ufb`. No pull request has been opened.
- Head commit: `34667a8b8` (plus the commit that adds this file).

| Path | What it holds |
|---|---|
| `apps/pdf-tool/core/src/lib.rs` | The PDF engine: `reduce`, `protect`, `unlock`, `process`. Tests at the bottom. |
| `apps/pdf-tool/src-tauri/src/main.rs` | Tauri commands: `list`, `pick`, `run`, `reveal`. One test. |
| `apps/pdf-tool/src-tauri/tauri.conf.json` | Window size (520×800, fixed), bundle targets, ad-hoc signing. |
| `apps/pdf-tool/ui/index.html` | The whole interface: plain HTML/CSS/JS, no build step, no npm. |
| `apps/pdf-tool/README.md` | User-facing install and build notes. |
| `.github/workflows/build-pdf-tool.yml` | Builds the `.dmg` on `macos-latest` on any push touching the app. |

## How to build and test

```sh
cd apps/pdf-tool/core && cargo test --release      # engine, 4 tests
cd ../src-tauri && cargo test                      # save/naming logic, 1 test
cd .. && npx -y @tauri-apps/cli@2 build            # on a Mac: produces the .dmg
```

On Linux the Tauri crate needs `libwebkit2gtk-4.1-dev` and `librsvg2-dev` to compile.

The installer comes from GitHub Actions: Actions → "Build PDF Tool (macOS)" → latest run →
Artifacts → `PDF-Tool-macOS` (a zip containing the `.dmg`). Latest good run:
https://github.com/bbenjyesq/printing-press-library/actions/runs/45467572875

## How it behaves

- **Saving:** no Save dialog. Each copy is written beside its original as `name-small.pdf`,
  `name-protected.pdf`, `name-small-protected.pdf` or `name-unlocked.pdf`. If the name is
  taken, ` 2`, ` 3`… is appended (`create_new`, so nothing is ever overwritten).
- **Reduce:** only images change. Each is resampled to 200 / 130 / 96 dpi at the size it is
  actually drawn on the page (found by walking page content streams and nested forms),
  re-encoded as progressive JPEG with optimized Huffman tables and 4:2:0 chroma, at quality
  75 / 50 / 32. Near-grey RGB images are stored as grey. A result that is not smaller than
  the input is reported as "already about as small as it can get".
- **Reduce + password:** a file that cannot be shrunk still gets its password.
- **Remove password:** decrypts while loading (`LoadOptions::with_password`). Do not switch
  to `Document::decrypt()` after a plain load; with lopdf 0.47 that left the page tree
  unreadable.
- **Batch:** the UI calls `run` once per file, so progress and per-file errors are free.
  One password applies to the whole batch.

## What has been verified, and how

- Engine tests pass on Linux and on GitHub's Apple-silicon runner.
- Output cross-checked with qpdf and Poppler: reduced files are valid and keep their text;
  protected files open only with the password; unlocked files are unencrypted and clean.
- Remove password tested against qpdf-encrypted files at 40-bit RC4, 128-bit RC4, 128-bit
  AES and 256-bit AES, with user and owner passwords.
- Compression measured on six generated fixtures (not committed). Balanced results: 300-dpi
  colour scan 10.0 MB → 559 KB; 150-dpi colour scan 1.4 MB → 563 KB; document with photos
  4.6 MB → 74 KB. Reduced scans were rendered and read for legibility.
- The window was launched on Linux under a virtual display, and the UI flows were exercised
  in a browser with the Tauri calls stubbed.

## What has NOT been verified

- **Nothing has been run by the developer on a real Mac.** The owner has installed it and
  got as far as the Gatekeeper prompt; whether they completed "Open Anyway" and used the app
  is not known.
- No output has been opened in Acrobat or Preview.
- No real client PDFs have been tested; the owner reported the first version's compression
  as "very modest" on their files, and the stronger settings have not had their feedback yet.
- Drag-and-drop, the native file picker and "Show in Finder" (`open -R`) only run on macOS
  and are untested.

## Known limits

- 1-bit black-and-white scans (CCITT/JBIG2) and text-only PDFs barely shrink.
- Skipped images: CMYK JPEGs, JPEG 2000, Indexed colour, anything not 8 bits per channel,
  images with `/Mask` or `/Decode`.
- Images the page walk does not find (pattern fills, soft masks) are assumed to fill a
  letter page; see the `shortcut:` comment in `shrink_image`.
- Reduce and Add password refuse files that are already encrypted (use Remove first).
- Remove password strips owner-only restrictions from a file that opens without a password,
  whatever is typed.
- No dark mode. When a batch has failures, the result links sit below the list and may need
  a scroll.

## Installing on the Mac (current friction)

The app is ad-hoc signed, not signed with an Apple Developer ID, so first launch shows
"Apple could not verify…". The fix is System Settings → Privacy & Security → Open Anyway, or
`xattr -cr "/Applications/PDF Tool.app"`. The app must be dragged into Applications first;
run from the mounted `.dmg` it will not appear in Spotlight. Removing the prompt for good
needs a paid Apple Developer account and notarization in the workflow.

## Open items

1. **UI UX Pro Max skill.** The owner asked for it twice. It is not installed in the
   workspace, and attaching `nextlevelbuilder/ui-ux-pro-max-skill` was blocked by the
   session's repository scope. The interface was designed with the taste-skill minimalist
   skill instead. Needs the owner to install the skill or approve access.
2. **Owner feedback on compression** after the stronger settings.
3. **Spotlight** not finding the app; expected to resolve once it is in Applications.
4. Offered and not chosen: a "get it under N MB" target, merging PDFs, publishing the `.dmg`
   as a GitHub Release (one-click download, no zip), restoring a Save dialog for single files.
5. Deliberately out of scope: redaction, Bates stamping, OCR, metadata scrubbing,
   print/copy restrictions.

## Build notes

- `macos-14` runners cancelled instantly; the workflow uses `macos-latest`.
- One run sat queued 15 minutes and was cancelled by GitHub for lack of a Mac runner; the
  next push built normally. Mac minutes bill at 10× on private repositories.
- The work was done under the "ponytail" skill: smallest complete change, no speculative
  features, one test per piece of non-trivial logic.
