#[macro_use]
extern crate num_derive;
#[macro_use]
extern crate serde_big_array;

use bincode;
use flate2::write::{ZlibDecoder, ZlibEncoder};
use serde::{Deserialize, Serialize};
use std::io;
use std::io::Write;

// Version of protocol. Increment this on change.
pub const VERSION: u16 = 0;

// Byte used for marking the start of a message
const HEADER: u8 = 174;

mod payloads;
pub use payloads::*;

// Types of messages passed through websockets
#[repr(u8)]
#[serde(untagged)]
#[derive(
    Serialize,
    Deserialize,
    FromPrimitive,
    Copy,
    Clone,
    Eq,
    PartialEq,
    std::fmt::Debug,
)]
pub enum MessageType {
    Handshake = 1,
    Synchronize,
    CreateThread,
    InsertPost,
}

// Appends 0 after HEADER byte to distinguish it from a message start
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
        let src_len = buf.len();

        while buf.len() > 0 {
            match buf.iter().position(|b| *b == HEADER) {
                Some(i) => {
                    self.w.write_all(&buf[..i])?;
                    self.w.write_all(&[HEADER, 0])?;
                    buf = &buf[i + 1..];
                }
                None => {
                    self.w.write_all(&buf)?;
                    return Ok(src_len);
                }
            }
        }

        Ok(src_len)
    }

    fn flush(&mut self) -> io::Result<()> {
        self.w.flush()
    }
}

// Streaming message set encoder
pub struct Encoder<W: Write> {
    w: ZlibEncoder<W>,
}

impl<W: Write> Encoder<W> {
    // Create new encoder for building message streams, which will have its
    // output written to the passed output stream.
    pub fn new(w: W) -> Encoder<W> {
        Self {
            w: ZlibEncoder::new(w, flate2::Compression::default()),
        }
    }

    // Flush any pending data to output stream
    pub fn flush(&mut self) -> io::Result<()> {
        self.w.flush()
    }

    // Write message to underlying writer
    pub fn write_message<T: Serialize>(
        &mut self,
        t: MessageType,
        payload: &T,
    ) -> io::Result<()> {
        self.w.write_all(&[HEADER, t as u8])?;
        bincode::serialize_into(&mut Escaper::new(&mut self.w), payload)
            .map_err(|err| {
                io::Error::new(io::ErrorKind::Other, err.to_string())
            })
    }

    // Consumes this encoder, flushing the output stream and returning the
    // underlying writer
    pub fn finish(self) -> io::Result<W> {
        self.w.finish()
    }
}

// Decompresses and decodes message batch.
pub struct Decoder {
    splitter: MessageSplitter,
    off: usize,
}

impl Decoder {
    // Create new decoder for reading the passed stream
    pub fn new(r: &[u8]) -> Result<Self, io::Error> {
        let mut zd = ZlibDecoder::new(MessageSplitter::new());
        zd.write_all(r)?;
        Ok(Self {
            splitter: zd.finish()?,
            off: 0,
        })
    }

    // Return next message type, if any.
    // Returns None, if entire message stream has been consumed.
    //
    // This method does not advance the decoder. Either read_next() or
    // skip_next() need to be called to advance it.
    pub fn peek_type(&mut self) -> Option<MessageType> {
        self.splitter.message_types.get(self.off).map(|t| *t)
    }

    // Skip reading next message and advance the decoder
    pub fn skip_next(&mut self) {
        self.off += 1;
    }

    // Read, decode and return next message payload from stream
    pub fn read_next<'a, 's: 'a, T: Deserialize<'a>>(
        &'s mut self,
    ) -> io::Result<T> {
        match self.splitter.message_starts.get(self.off) {
            None => Err(io::Error::from(io::ErrorKind::NotFound)),
            Some(i) => {
                let to = *self
                    .splitter
                    .message_starts
                    .get(self.off + 1)
                    .unwrap_or(&self.splitter.buf.len());
                let buf = &self.splitter.buf[*i..to];
                let res = bincode::deserialize(buf).map_err(|err| {
                    io::Error::new(io::ErrorKind::InvalidData, err.to_string())
                })?;
                self.off += 1;
                Ok(res)
            }
        }
    }
}

// Splits and unescapes messages
#[derive(Debug)]
struct MessageSplitter {
    buf: Vec<u8>,
    message_starts: Vec<usize>,
    message_types: Vec<MessageType>,
}

impl MessageSplitter {
    fn new() -> Self {
        Self {
            buf: Vec::with_capacity(1 << 10),
            message_starts: Vec::new(),
            message_types: Vec::new(),
        }
    }
}

impl Write for MessageSplitter {
    fn write(&mut self, mut buf: &[u8]) -> io::Result<usize> {
        fn err_invalid_data<T>(msg: impl Into<String>) -> io::Result<T> {
            Err(io::Error::new(io::ErrorKind::InvalidData, msg.into()))
        }

        let src_len = buf.len();

        while buf.len() > 0 {
            match buf.iter().position(|b| *b == HEADER) {
                Some(i) => {
                    if i == buf.len() - 1 {
                        return err_invalid_data(
                            "buffer ends with header byte",
                        );
                    }

                    let next = buf[i + 1];
                    if next != 0 {
                        // Found next message
                        match num::FromPrimitive::from_u8(next) {
                            Some(typ) => {
                                self.message_starts.push(self.buf.len() + i);
                                self.message_types.push(typ);
                                self.buf.extend(&buf[..i]);
                            }
                            None => {
                                return err_invalid_data(format!(
                                    "invalid message type: {}",
                                    buf[1],
                                ))
                            }
                        };
                    } else {
                        // Escaped byte
                        self.buf.extend(&buf[..i + 1]);
                    }
                    buf = &buf[i + 2..];
                }
                None => {
                    self.buf.extend(buf);
                    return Ok(src_len);
                }
            }
        }

        Ok(src_len)
    }

    fn flush(&mut self) -> io::Result<()> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use serde::{Deserialize, Serialize};

    type Result = std::result::Result<(), Box<dyn std::error::Error>>;

    #[derive(Serialize, Deserialize, Debug, PartialEq, Eq)]
    struct SimpleMessage<'a> {
        a: u64,
        b: &'a str,
        buf: &'a [u8],
    }

    #[test]
    fn simple_message_stream() -> Result {
        const WITH_HEADER: [u8; 6] = [1, 2, 3, super::HEADER, 1, 3];

        let mut enc = super::Encoder::new(Vec::<u8>::new());
        for i in 1..=4 {
            enc.write_message(
                num::FromPrimitive::from_u64(i).unwrap(),
                &SimpleMessage {
                    a: i,
                    b: &i.to_string(),
                    // Include HEADER for escape testing
                    buf: &WITH_HEADER,
                },
            )?;
        }
        let buf = &enc.finish()?;

        let mut dec = super::Decoder::new(buf)?;
        for i in 1..=4 {
            assert_eq!(dec.peek_type(), num::FromPrimitive::from_u64(i));
            let res: SimpleMessage = dec.read_next()?;
            assert_eq!(
                res,
                SimpleMessage {
                    a: i,
                    b: &i.to_string(),
                    // Include HEADER for escape testing
                    buf: &WITH_HEADER,
                },
            );
        }

        Ok(())
    }
}
