use anyhow::Result;
use crossterm::{
    event::{self, Event, KeyCode},
    execute,
    terminal::{EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode},
};
use ratatui::{Terminal, backend::CrosstermBackend, widgets::{Block, Borders, Paragraph}};
use std::{io, time::Duration};

fn main() -> Result<()> {
    let frames = rust_shared::load_frames(rust_shared::DEFAULT_LAT, rust_shared::DEFAULT_LON, rust_shared::DEFAULT_ZOOM)?;
    let img = rust_shared::composite(&frames[0])?;
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;
    loop {
        terminal.draw(|f| {
            let area = f.area();
            let (w, h) = rust_shared::fit_square_cells(area.width.saturating_sub(4), area.height.saturating_sub(4));
            let text = render_halfblocks(&img, w, h);
            let p = Paragraph::new(text).block(Block::default().title("rust-ratatui-halfblocks").borders(Borders::ALL));
            f.render_widget(p, area);
        })?;
        if event::poll(Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if matches!(key.code, KeyCode::Char('q') | KeyCode::Esc) {
                    break;
                }
            }
        }
    }
    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    Ok(())
}

fn render_halfblocks(img: &image::DynamicImage, width: u16, height: u16) -> String {
    use image::GenericImageView;
    let (iw, ih) = img.dimensions();
    let mut out = String::new();
    for row in 0..height {
        let upper_y = (u32::from(row) * 2 * ih) / (u32::from(height) * 2);
        let lower_y = ((u32::from(row) * 2 + 1) * ih) / (u32::from(height) * 2);
        for col in 0..width {
            let x = (u32::from(col) * iw) / u32::from(width);
            let upper = img.get_pixel(x, upper_y);
            let lower = img.get_pixel(x, lower_y);
            out.push_str(&format!(
                "\x1b[38;2;{};{};{}m\x1b[48;2;{};{};{}m▀",
                upper[0], upper[1], upper[2], lower[0], lower[1], lower[2]
            ));
        }
        out.push_str("\x1b[0m\n");
    }
    out
}
