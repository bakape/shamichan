use bincode;
use compress::lz4::Encoder as LZ4Encoder;
use serde::{Deserialize, Serialize};
use std::io;
use std::io::Write;

// Version of protocol. Increment this on change.
pub const VERSION: u32 = 0;

// Sequence used for marking the start of a message
const HEADER: [u8; 2] = [254, 255];

// Types of messages passed through websockets
#[repr(u8)]
#[serde(untagged)]
#[derive(Serialize, Deserialize)]
pub enum MessageType {
    Invalid = 1,
    Composite,
    Synchronize,
    CreateThread,
    InsertPost,
}

// Escapes [254, 255] -> [254, 255, 0]
struct Escaper<W: Write> {
    w: W,
}

impl<W: Write> Escaper<W> {
    fn new(w: W) -> Escaper<W> {
        Self { w: w }
    }
}

impl<W: Write> Write for Escaper<W> {
    fn write(&mut self, mut buf: &[u8]) -> io::Result<usize> {
        while buf.len() > 0 {
            let mut to = buf.len();
            for (i, b) in buf.iter().enumerate() {
                if *b == HEADER[0]
                    && i != buf.len() - 1
                    && buf[i + 1] == HEADER[1]
                {
                    to = i;
                    break;
                }
            }

            self.w.write_all(&buf[..to])?;

            if to == buf.len() {
                return Ok(buf.len());
            } else {
                self.w.write_all(&[HEADER[0], HEADER[1], 0])?;
                buf = &buf[..to + 2];
            }
        }

        Ok(buf.len())
    }

    fn flush(&mut self) -> io::Result<()> {
        self.w.flush()
    }
}

// Streaming message set encoder
pub struct Encoder<W: Write> {
    lz4_w: LZ4Encoder<W>,
}

impl<W: Write> Encoder<W> {
    // Create new encoder for building message streams, which will have its
    // output written to the passed output stream.
    pub fn new(w: W) -> Encoder<W> {
        Self {
            lz4_w: LZ4Encoder::new(w),
        }
    }

    // Flush any pending data to output stream
    pub fn flush(&mut self) -> io::Result<()> {
        self.lz4_w.flush()
    }

    // Write message to underlying writer
    pub fn write_message<T: Serialize>(
        &mut self,
        t: MessageType,
        payload: &T,
    ) -> Result<(), String> {
        self.lz4_w
            .write_all(&[254, 255, t as u8])
            .map_err(|err| format!("{}", err))?;

        bincode::serialize_into(&mut Escaper::new(&mut self.lz4_w), payload)
            .map_err(|err| format!("{}", err))
    }
}
