#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::io::Write;
use std::path::{Path, PathBuf};
use tauri_plugin_dialog::DialogExt;

#[derive(serde::Serialize)]
struct Saved {
    path: PathBuf,
    before: u64,
    after: u64,
}

fn is_pdf(path: &Path) -> bool {
    path.extension().is_some_and(|e| e.eq_ignore_ascii_case("pdf"))
}

/// The PDFs among `paths` with their sizes; a folder contributes the PDFs directly inside it.
#[tauri::command]
fn list(paths: Vec<PathBuf>) -> Vec<(PathBuf, u64)> {
    let mut files: Vec<PathBuf> = Vec::new();
    for p in paths {
        match std::fs::read_dir(&p) {
            Ok(dir) => files.extend(dir.flatten().map(|e| e.path())),
            Err(_) => files.push(p),
        }
    }
    files.sort();
    files.dedup();
    files
        .into_iter()
        .filter(|p| is_pdf(p))
        .filter_map(|p| Some((std::fs::metadata(&p).ok().filter(|m| m.is_file())?.len(), p)))
        .map(|(len, p)| (p, len))
        .collect()
}

#[tauri::command]
async fn pick(app: tauri::AppHandle) -> Vec<PathBuf> {
    let picked = app.dialog().file().add_filter("PDF", &["pdf"]).blocking_pick_files();
    picked.into_iter().flatten().filter_map(|f| f.into_path().ok()).collect()
}

/// Reduces and/or protects one PDF and saves the copy beside it as "<name>-small.pdf",
/// "<name>-protected.pdf" or "<name>-small-protected.pdf". Never overwrites anything: if
/// that name is taken, " 2", " 3"… is appended.
#[tauri::command]
async fn run(path: PathBuf, level: Option<String>, password: Option<String>) -> Result<Saved, String> {
    if !is_pdf(&path) {
        return Err("That is not a PDF file.".into());
    }
    let input = std::fs::read(&path).map_err(|e| format!("Could not open the file ({e})."))?;
    let output = pdfcore::process(&input, level.as_deref(), password.as_deref())?;
    let stem = path.file_stem().unwrap_or_default().to_string_lossy();
    let suffix = match (level.is_some() && output.len() < input.len(), password.is_some()) {
        (true, true) => "-small-protected",
        (true, false) => "-small",
        (false, _) => "-protected",
    };
    for n in 1..1000 {
        let count = if n == 1 { String::new() } else { format!(" {n}") };
        let dest = path.with_file_name(format!("{stem}{suffix}{count}.pdf"));
        // create_new is the "never overwrite" guarantee.
        match std::fs::OpenOptions::new().write(true).create_new(true).open(&dest) {
            Ok(mut file) => {
                file.write_all(&output).map_err(|e| format!("Could not save the file ({e})."))?;
                return Ok(Saved { path: dest, before: input.len() as u64, after: output.len() as u64 });
            }
            Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(e) => return Err(format!("Could not save next to the original ({e}).")),
        }
    }
    Err("Too many copies of this file already exist.".into())
}

#[tauri::command]
fn reveal(path: PathBuf) -> Result<(), String> {
    std::process::Command::new("open").arg("-R").arg(path).spawn().map(drop).map_err(|e| e.to_string())
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![list, pick, run, reveal])
        .run(tauri::generate_context!())
        .expect("error while running PDF Tool");
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A minimal one-page PDF with a correct xref table.
    fn tiny_pdf() -> Vec<u8> {
        let objects = [
            "<< /Type /Catalog /Pages 2 0 R >>",
            "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
            "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>",
        ];
        let mut pdf = String::from("%PDF-1.4\n");
        let mut xref = String::from("xref\n0 4\n0000000000 65535 f \n");
        for (i, body) in objects.iter().enumerate() {
            xref += &format!("{:010} 00000 n \n", pdf.len());
            pdf += &format!("{} 0 obj\n{body}\nendobj\n", i + 1);
        }
        let start = pdf.len();
        (pdf + &xref + &format!("trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n{start}\n%%EOF\n")).into_bytes()
    }

    #[test]
    fn saves_beside_the_original_without_overwriting() {
        let dir = std::env::temp_dir().join(format!("pdf-tool-test-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let original = dir.join("Exhibit A.PDF");
        std::fs::write(&original, tiny_pdf()).unwrap();
        std::fs::write(dir.join("notes.txt"), "x").unwrap();

        // A dropped folder lists only its PDFs.
        let listed = list(vec![dir.clone()]);
        assert_eq!(listed.len(), 1);
        assert_eq!(listed[0].0, original);

        let go = |level: Option<&str>| {
            tauri::async_runtime::block_on(run(original.clone(), level.map(Into::into), Some("pw".into())))
        };
        let first = go(None).unwrap();
        let second = go(Some("balanced")).unwrap(); // nothing to shrink, still protected
        assert_eq!(first.path, dir.join("Exhibit A-protected.pdf"));
        assert_eq!(second.path, dir.join("Exhibit A-protected 2.pdf"));
        assert_eq!(std::fs::read(&original).unwrap(), tiny_pdf(), "original untouched");
        assert!(tauri::async_runtime::block_on(run(original.clone(), Some("balanced".into()), None)).is_err());
        assert!(tauri::async_runtime::block_on(run(dir.join("notes.txt"), None, Some("pw".into()))).is_err());
        std::fs::remove_dir_all(&dir).unwrap();
    }
}
