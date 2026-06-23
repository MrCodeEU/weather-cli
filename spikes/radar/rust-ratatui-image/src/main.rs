use anyhow::Result;
use crossterm::{
    event::{self, Event, KeyCode},
    execute,
    terminal::{EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode},
};
use ratatui::{
    Terminal,
    backend::CrosstermBackend,
    layout::{Constraint, Direction, Layout},
    style::{Color, Modifier, Style},
    widgets::{Block, Borders, Paragraph},
};
use ratatui_image::{Resize, StatefulImage, picker::Picker, protocol::StatefulProtocol};
use std::{io, time::Duration};

fn main() -> Result<()> {
    let frames = rust_shared::load_frames(rust_shared::DEFAULT_LAT, rust_shared::DEFAULT_LON, rust_shared::DEFAULT_ZOOM)?;
    let frame = frames.first().expect("rainviewer returns at least one frame");
    let img = rust_shared::composite(frame)?;
    let picker = Picker::from_query_stdio().unwrap_or_else(|_| Picker::from_fontsize((8, 16)));
    let mut image = picker.new_resize_protocol(img);

    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let result = run(&mut terminal, frame, &mut image);

    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    result
}

fn run(
    terminal: &mut Terminal<CrosstermBackend<io::Stdout>>,
    frame: &rust_shared::Frame,
    image: &mut StatefulProtocol,
) -> Result<()> {
    loop {
        terminal.draw(|f| {
            let area = f.area();
            let rows = Layout::default()
                .direction(Direction::Vertical)
                .constraints([
                    Constraint::Length(3),
                    Constraint::Min(1),
                    Constraint::Length(2),
                ])
                .split(area);

            let title = Paragraph::new(format!(
                "rust-ratatui-image - {} - {} - q/Esc quit",
                frame.kind,
                frame_time(frame.time)
            ))
            .style(Style::default().fg(Color::Yellow).add_modifier(Modifier::BOLD))
            .block(Block::default().borders(Borders::BOTTOM));
            f.render_widget(title, rows[0]);

            let image_area = Block::default()
                .title("Ratatui StatefulImage")
                .borders(Borders::ALL)
                .inner(rows[1]);
            f.render_stateful_widget(
                StatefulImage::new().resize(Resize::Fit(None)),
                image_area,
                image,
            );

            let footer = Paragraph::new("Uses ratatui-image protocol picker: Kitty, Sixel, iTerm2, or halfblocks fallback.")
                .style(Style::default().fg(Color::DarkGray))
                .block(Block::default().borders(Borders::TOP));
            f.render_widget(footer, rows[2]);
        })?;

        if let Some(result) = image.last_encoding_result() {
            result?;
        }

        if event::poll(Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                if matches!(key.code, KeyCode::Char('q') | KeyCode::Esc) {
                    break;
                }
            }
        }
    }
    Ok(())
}

fn frame_time(timestamp: i64) -> String {
    let Some(dt) = chrono::DateTime::from_timestamp(timestamp, 0) else {
        return timestamp.to_string();
    };
    dt.format("%H:%M").to_string()
}
