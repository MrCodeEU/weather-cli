use anyhow::{Context, Result};
use image::{DynamicImage, GenericImage, GenericImageView, ImageBuffer, Rgba};
use serde::Deserialize;

pub const DEFAULT_LAT: f64 = 48.2082;
pub const DEFAULT_LON: f64 = 16.3738;
pub const DEFAULT_ZOOM: u8 = 5;

#[derive(Clone, Debug)]
pub struct Frame {
    pub time: i64,
    pub kind: String,
    pub url: String,
    pub base_url: String,
}

#[derive(Deserialize)]
struct RainViewer {
    host: String,
    radar: Radar,
}

#[derive(Deserialize)]
struct Radar {
    past: Vec<RadarFrame>,
    nowcast: Vec<RadarFrame>,
}

#[derive(Deserialize)]
struct RadarFrame {
    time: i64,
    path: String,
}

pub fn load_frames(lat: f64, lon: f64, zoom: u8) -> Result<Vec<Frame>> {
    let client = http_client()?;
    let body = client
        .get("https://api.rainviewer.com/public/weather-maps.json")
        .send()
        .context("fetch rainviewer metadata")?
        .error_for_status()
        .context("rainviewer metadata status")?
        .text()
        .context("read rainviewer metadata")?;
    let raw: RainViewer = serde_json::from_str(&body).context("parse rainviewer metadata")?;
    let (_, x, y) = tile_for(lat, lon, zoom);
    let base_url = format!("https://tile.openstreetmap.org/{zoom}/{x}/{y}.png");
    let mut frames = Vec::with_capacity(raw.radar.past.len() + raw.radar.nowcast.len());
    add_frames(&mut frames, &raw.host, "past", &raw.radar.past, zoom, x, y, &base_url);
    add_frames(&mut frames, &raw.host, "nowcast", &raw.radar.nowcast, zoom, x, y, &base_url);
    Ok(frames)
}

pub fn composite(frame: &Frame) -> Result<DynamicImage> {
    let base = fetch_image(&frame.base_url).context("fetch basemap")?;
    let overlay = fetch_image(&frame.url).context("fetch radar overlay")?;
    let (w, h) = base.dimensions();
    let mut canvas = DynamicImage::ImageRgba8(ImageBuffer::from_pixel(w, h, Rgba([0, 0, 0, 0])));
    image::imageops::overlay(&mut canvas, &base, 0, 0);
    image::imageops::overlay(&mut canvas, &overlay, 0, 0);
    draw_marker(&mut canvas);
    Ok(canvas)
}

pub fn fit_square_cells(width: u16, height: u16) -> (u16, u16) {
    let mut w = width.max(20);
    let mut h = (w / 2).max(8);
    if h > height.max(8) {
        h = height.max(8);
        w = h * 2;
    }
    (w, h)
}

fn add_frames(
    out: &mut Vec<Frame>,
    host: &str,
    kind: &str,
    frames: &[RadarFrame],
    zoom: u8,
    x: u32,
    y: u32,
    base_url: &str,
) {
    for frame in frames {
        out.push(Frame {
            time: frame.time,
            kind: kind.to_string(),
            url: format!("{host}{}/256/{zoom}/{x}/{y}/2/1_1.png", frame.path),
            base_url: base_url.to_string(),
        });
    }
}

fn fetch_image(url: &str) -> Result<DynamicImage> {
    let client = http_client()?;
    let bytes = client
        .get(url)
        .send()
        .with_context(|| format!("fetch {url}"))?
        .error_for_status()
        .with_context(|| format!("status {url}"))?
        .bytes()
        .with_context(|| format!("read {url}"))?;
    image::load_from_memory(&bytes).with_context(|| format!("decode {url}"))
}

fn http_client() -> Result<reqwest::blocking::Client> {
    reqwest::blocking::Client::builder()
        .user_agent("weather-cli-radar-spike/0.1")
        .build()
        .context("build http client")
}

fn tile_for(lat: f64, lon: f64, zoom: u8) -> (u8, u32, u32) {
    let lat_rad = lat.to_radians();
    let n = 2_f64.powi(i32::from(zoom));
    let x = ((lon + 180.0) / 360.0 * n).floor() as u32;
    let y = ((1.0 - ((lat_rad.tan() + 1.0 / lat_rad.cos()).ln() / std::f64::consts::PI)) / 2.0 * n).floor() as u32;
    (zoom, x, y)
}

fn draw_marker(img: &mut DynamicImage) {
    let (w, h) = img.dimensions();
    if w == 0 || h == 0 {
        return;
    }
    let cx = (w / 2) as i32;
    let cy = (h / 2) as i32;
    for d in -8..=8 {
        set_safe(img, cx + d, cy, [255, 255, 255, 255]);
        set_safe(img, cx, cy + d, [255, 255, 255, 255]);
    }
    for d in -5..=5 {
        set_safe(img, cx + d, cy, [255, 32, 32, 255]);
        set_safe(img, cx, cy + d, [255, 32, 32, 255]);
    }
}

fn set_safe(img: &mut DynamicImage, x: i32, y: i32, color: [u8; 4]) {
    if x < 0 || y < 0 {
        return;
    }
    let (w, h) = img.dimensions();
    let x = x as u32;
    let y = y as u32;
    if x < w && y < h {
        img.put_pixel(x, y, Rgba(color));
    }
}
