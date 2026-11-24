//! PDF work for the app: shrink a PDF by recompressing its images, or lock it with a password.

use image::{codecs::jpeg::JpegEncoder, imageops::FilterType, DynamicImage, GrayImage, RgbImage};
use lopdf::encryption::crypt_filters::{Aes256CryptFilter, CryptFilter};
use lopdf::{Dictionary, Document, EncryptionState, EncryptionVersion, Object, Permissions, Stream};
use rayon::prelude::*;
use std::{collections::BTreeMap, sync::Arc};

fn load(input: &[u8]) -> Result<Document, String> {
    let doc = Document::load_mem(input).map_err(|e| format!("This file could not be read as a PDF ({e})."))?;
    if doc.is_encrypted() || doc.was_encrypted() {
        return Err("This PDF is already password-protected. Remove its password first.".into());
    }
    Ok(doc)
}

/// (longest side in pixels, JPEG quality) for "light" | "balanced" | "strong".
// shortcut: caps pixels instead of computing each image's printed DPI — exact for
// full-page letter scans (200/150/100 dpi), generous for anything smaller. Upgrade if
// large-format pages (exhibits, plans) come out too soft.
fn settings(level: &str) -> Result<(u32, u8), String> {
    match level {
        "light" => Ok((2200, 75)),
        "balanced" => Ok((1650, 60)),
        "strong" => Ok((1100, 45)),
        _ => Err(format!("Unknown level: {level}")),
    }
}

/// Colour channels of an image we know how to re-encode as JPEG (gray or RGB), else None.
fn channels(doc: &Document, dict: &Dictionary) -> Option<u8> {
    let cs = dict.get_deref(b"ColorSpace", doc).ok()?;
    if let Ok(name) = cs.as_name() {
        return match name {
            b"DeviceGray" => Some(1),
            b"DeviceRGB" => Some(3),
            _ => None,
        };
    }
    let arr = cs.as_array().ok()?;
    if arr.first()?.as_name().ok()? != b"ICCBased" {
        return None;
    }
    let (_, profile) = doc.dereference(arr.get(1)?).ok()?;
    match profile.as_stream().ok()?.dict.get(b"N").ok()?.as_i64().ok()? {
        1 => Some(1),
        3 => Some(3),
        _ => None,
    }
}

fn filter_name(dict: &Dictionary) -> Option<&[u8]> {
    match dict.get(b"Filter").ok()? {
        Object::Name(n) => Some(n),
        Object::Array(a) if a.len() == 1 => a[0].as_name().ok(),
        _ => None,
    }
}

/// A smaller JPEG version of one image stream, or None to leave it untouched.
fn shrink_image(doc: &Document, s: &Stream, max_side: u32, quality: u8) -> Option<Stream> {
    let d = &s.dict;
    if d.get(b"Subtype").ok()?.as_name().ok()? != b"Image"
        || d.has(b"ImageMask")
        || d.has(b"Mask")
        || d.has(b"Decode")
        || d.get_deref(b"BitsPerComponent", doc).ok()?.as_i64().ok()? != 8
    {
        return None;
    }
    let n = channels(doc, d)?;
    let img = match filter_name(d)? {
        b"DCTDecode" => {
            let img = image::load_from_memory_with_format(&s.content, image::ImageFormat::Jpeg).ok()?;
            // CMYK and friends decode to something that no longer matches /ColorSpace.
            match (n, &img) {
                (1, DynamicImage::ImageLuma8(_)) | (3, DynamicImage::ImageRgb8(_)) => img,
                _ => return None,
            }
        }
        b"FlateDecode" => {
            let w = d.get_deref(b"Width", doc).ok()?.as_i64().ok()? as u32;
            let h = d.get_deref(b"Height", doc).ok()?.as_i64().ok()? as u32;
            let px = s.decompressed_content().ok()?;
            if n == 1 {
                DynamicImage::ImageLuma8(GrayImage::from_raw(w, h, px)?)
            } else {
                DynamicImage::ImageRgb8(RgbImage::from_raw(w, h, px)?)
            }
        }
        _ => return None,
    };
    let img = if img.width().max(img.height()) > max_side {
        img.resize(max_side, max_side, FilterType::Triangle)
    } else {
        img
    };
    let mut jpeg = Vec::new();
    img.write_with_encoder(JpegEncoder::new_with_quality(&mut jpeg, quality)).ok()?;
    if jpeg.len() >= s.content.len() {
        return None;
    }
    let mut dict = d.clone();
    dict.set("Width", img.width() as i64);
    dict.set("Height", img.height() as i64);
    dict.set("Filter", "DCTDecode");
    dict.remove(b"DecodeParms");
    Some(Stream::new(dict, jpeg))
}

/// Returns a smaller copy of the PDF. Text, fonts and vector art are untouched; only
/// images are downsampled and recompressed.
pub fn reduce(input: &[u8], level: &str) -> Result<Vec<u8>, String> {
    let (max_side, quality) = settings(level)?;
    let mut doc = load(input)?;
    let smaller: Vec<_> = doc
        .objects
        .par_iter()
        .filter_map(|(id, obj)| Some((*id, shrink_image(&doc, obj.as_stream().ok()?, max_side, quality)?)))
        .collect();
    for (id, stream) in smaller {
        doc.objects.insert(id, Object::Stream(stream));
    }
    doc.compress();
    let mut out = Vec::new();
    doc.save_modern(&mut out).map_err(|e| format!("Could not write the PDF ({e})."))?;
    if out.len() >= input.len() {
        return Err("This PDF is already about as small as it can get.".into());
    }
    Ok(out)
}

/// Returns a copy of the PDF that needs `password` to open (AES-256).
pub fn protect(input: &[u8], password: &str) -> Result<Vec<u8>, String> {
    if password.is_empty() {
        return Err("Enter a password.".into());
    }
    let mut doc = load(input)?;
    let key: [u8; 32] = rand::random();
    let filter: Arc<dyn CryptFilter> = Arc::new(Aes256CryptFilter);
    let state = EncryptionState::try_from(EncryptionVersion::V5 {
        encrypt_metadata: true,
        crypt_filters: BTreeMap::from([(b"StdCF".to_vec(), filter)]),
        file_encryption_key: &key,
        stream_filter: b"StdCF".to_vec(),
        string_filter: b"StdCF".to_vec(),
        owner_password: password,
        user_password: password,
        permissions: Permissions::all(),
    })
    .and_then(|state| doc.encrypt(&state).map(|_| state))
    .map_err(|e| format!("Could not encrypt the PDF ({e})."));
    state?;
    if !doc.trailer.has(b"ID") {
        // Encrypted files are expected to carry a file ID; some readers warn without one.
        let id = || Object::string_literal(rand::random::<[u8; 16]>().to_vec());
        doc.trailer.set("ID", vec![id(), id()]);
    }
    let mut out = Vec::new();
    doc.save_to(&mut out).map_err(|e| format!("Could not write the PDF ({e})."))?;
    Ok(out)
}

/// Reduce (when `level` is given), then protect (when `password` is given).
pub fn process(input: &[u8], level: Option<&str>, password: Option<&str>) -> Result<Vec<u8>, String> {
    let reduced = match (level.map(|l| reduce(input, l)), password) {
        (Some(Ok(out)), _) => Some(out),
        (Some(Err(e)), None) => return Err(e),
        // Already-compact files still get their password; an unreadable one fails in protect().
        (Some(Err(_)), Some(_)) | (None, _) => None,
    };
    let data = reduced.as_deref().unwrap_or(input);
    match password {
        Some(pw) => protect(data, pw),
        None => reduced.ok_or_else(|| "Nothing to do.".into()),
    }
}

#[cfg(test)]
#[allow(deprecated)]
mod tests {
    use super::*;
    use lopdf::{content::{Content, Operation}, dictionary};

    /// One page with a line of text and a noisy 2400x3000 RGB flate image (a fake scan).
    fn sample() -> Vec<u8> {
        let (w, h) = (2400u32, 3000u32);
        let mut seed = 1u32;
        let px: Vec<u8> = (0..w * h * 3)
            .map(|i| {
                seed = seed.wrapping_mul(1664525).wrapping_add(1013904223);
                ((i / 3 % w) / 10) as u8 ^ (seed >> 28) as u8
            })
            .collect();
        let mut doc = Document::with_version("1.5");
        let pages = doc.new_object_id();
        let mut img = Stream::new(
            dictionary! { "Type" => "XObject", "Subtype" => "Image", "Width" => w as i64, "Height" => h as i64,
            "ColorSpace" => "DeviceRGB", "BitsPerComponent" => 8 },
            px,
        );
        img.compress().unwrap();
        let img = doc.add_object(img);
        let font = doc.add_object(dictionary! { "Type" => "Font", "Subtype" => "Type1", "BaseFont" => "Helvetica" });
        let content = Content {
            operations: vec![
                Operation::new("q", vec![]),
                Operation::new("cm", vec![612.into(), 0.into(), 0.into(), 792.into(), 0.into(), 0.into()]),
                Operation::new("Do", vec!["Im1".into()]),
                Operation::new("Q", vec![]),
                Operation::new("BT", vec![]),
                Operation::new("Tf", vec!["F1".into(), 24.into()]),
                Operation::new("Td", vec![72.into(), 700.into()]),
                Operation::new("Tj", vec![Object::string_literal("Privileged and confidential")]),
                Operation::new("ET", vec![]),
            ],
        };
        let content = doc.add_object(Stream::new(dictionary! {}, content.encode().unwrap()));
        let page = doc.add_object(dictionary! {
            "Type" => "Page", "Parent" => pages, "Contents" => content,
            "MediaBox" => vec![0.into(), 0.into(), 612.into(), 792.into()],
            "Resources" => dictionary! { "XObject" => dictionary! { "Im1" => img }, "Font" => dictionary! { "F1" => font } },
        });
        doc.objects.insert(pages, Object::Dictionary(dictionary! { "Type" => "Pages", "Kids" => vec![page.into()], "Count" => 1 }));
        let catalog = doc.add_object(dictionary! { "Type" => "Catalog", "Pages" => pages });
        doc.trailer.set("Root", catalog);
        let mut out = Vec::new();
        doc.save_to(&mut out).unwrap();
        out
    }

    #[test]
    fn reduce_shrinks_and_keeps_text() {
        let input = sample();
        let out = reduce(&input, "balanced").unwrap();
        assert!(out.len() < input.len() / 2, "{} -> {}", input.len(), out.len());
        let doc = Document::load_mem(&out).unwrap();
        assert!(doc.extract_text(&[1]).unwrap().contains("Privileged and confidential"));
        let widths: Vec<i64> = doc.objects.values().filter_map(|o| o.as_stream().ok()?.dict.get(b"Width").ok()?.as_i64().ok()).collect();
        assert_eq!(widths, [1320]); // 2400x3000 capped to a 1650 long side
        assert!(reduce(&out, "light").is_err(), "nothing left to gain");
        if std::env::var("PDFCORE_DUMP").is_ok() {
            std::fs::write("/tmp/pdfcore/in.pdf", &input).unwrap();
            std::fs::write("/tmp/pdfcore/small.pdf", &out).unwrap();
            std::fs::write("/tmp/pdfcore/locked.pdf", protect(&input, "pässword 1").unwrap()).unwrap();
        }
    }

    #[test]
    fn process_reduces_then_protects() {
        let input = sample();
        let out = process(&input, Some("balanced"), Some("pw")).unwrap();
        assert!(out.len() < input.len() / 2);
        assert!(Document::load_mem_with_password(&out, "nope").is_err());
        let doc = Document::load_mem_with_password(&out, "pw").unwrap();
        assert!(doc.extract_text(&[1]).unwrap().contains("Privileged and confidential"));
        // Already compact: reduce alone is an error, but with a password the file is still protected.
        let small = reduce(&input, "balanced").unwrap();
        assert!(process(&small, Some("light"), None).is_err());
        let locked = process(&small, Some("light"), Some("pw")).unwrap();
        assert!(Document::load_mem_with_password(&locked, "pw").is_ok());
        assert!(process(b"not a pdf", Some("light"), Some("pw")).is_err());
        if std::env::var("PDFCORE_DUMP").is_ok() {
            std::fs::write("/tmp/pdfcore/both.pdf", &out).unwrap();
        }
    }

    #[test]
    fn protect_requires_the_password() {
        let out = protect(&sample(), "correct horse").unwrap();
        assert!(Document::load_mem_with_password(&out, "wrong").is_err());
        let doc = Document::load_mem_with_password(&out, "correct horse").unwrap();
        assert!(doc.extract_text(&[1]).unwrap().contains("Privileged and confidential"));
        assert!(protect(&out, "again").is_err(), "already protected");
        assert!(protect(&sample(), "").is_err());
    }
}
