use anyhow::Result;

fn main() -> Result<()> {
    let frames = rust_shared::load_frames(rust_shared::DEFAULT_LAT, rust_shared::DEFAULT_LON, rust_shared::DEFAULT_ZOOM)?;
    let img = rust_shared::composite(&frames[0])?;
    println!("rust-viuer · {} · {}", frames[0].kind, frames[0].time);
    let conf = viuer::Config {
        transparent: false,
        ..Default::default()
    };
    viuer::print(&img, &conf)?;
    Ok(())
}
