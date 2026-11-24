#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::path::{Path, PathBuf};
use tauri_plugin_dialog::DialogExt;

#[derive(serde::Serialize)]
struct Saved {
    name: String,
    before: u64,
    after: u64,
}

fn is_pdf(path: &Path) -> bool {
    path.extension().is_some_and(|e| e.eq_ignore_ascii_case("pdf"))
}

/// Size in bytes of a PDF the user picked or dropped.
#[tauri::command]
fn size(path: PathBuf) -> Result<u64, String> {
    if !is_pdf(&path) {
        return Err("That is not a PDF file.".into());
    }
    std::fs::metadata(&path).map(|m| m.len()).map_err(|e| e.to_string())
}

#[tauri::command]
async fn pick(app: tauri::AppHandle) -> Option<PathBuf> {
    app.dialog().file().add_filter("PDF", &["pdf"]).blocking_pick_file()?.into_path().ok()
}

/// Runs `action` ("reduce" with a level, or "protect" with a password), then asks where to
/// save the copy. Ok(None) means the user cancelled the save dialog. The original is never
/// modified.
#[tauri::command]
async fn run(app: tauri::AppHandle, path: PathBuf, action: String, arg: String) -> Result<Option<Saved>, String> {
    if !is_pdf(&path) {
        return Err("That is not a PDF file.".into());
    }
    let input = std::fs::read(&path).map_err(|e| format!("Could not open the file ({e})."))?;
    let (output, suffix) = match action.as_str() {
        "reduce" => (pdfcore::reduce(&input, &arg)?, "small"),
        "protect" => (pdfcore::protect(&input, &arg)?, "protected"),
        _ => return Err(format!("Unknown action: {action}")),
    };
    let stem = path.file_stem().unwrap_or_default().to_string_lossy();
    let mut dialog = app.dialog().file().add_filter("PDF", &["pdf"]).set_file_name(format!("{stem}-{suffix}.pdf"));
    if let Some(dir) = path.parent() {
        dialog = dialog.set_directory(dir);
    }
    let Some(dest) = dialog.blocking_save_file() else { return Ok(None) };
    let dest = dest.into_path().map_err(|e| e.to_string())?;
    if dest == path {
        return Err("Choose a different name so the original is kept.".into());
    }
    std::fs::write(&dest, &output).map_err(|e| format!("Could not save the file ({e})."))?;
    Ok(Some(Saved {
        name: dest.file_name().unwrap_or_default().to_string_lossy().into(),
        before: input.len() as u64,
        after: output.len() as u64,
    }))
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![size, pick, run])
        .run(tauri::generate_context!())
        .expect("error while running PDF Tool");
}
